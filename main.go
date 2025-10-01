package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/AlecAivazis/survey/v2"
	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v74/github"
	"github.com/jamietanna/gh-discussion/internal/discussion"
	"github.com/jamietanna/gh-discussion/internal/discussionform"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "discussion",
		Usage: "Interact with GitHub Discussions",
		Commands: []*cli.Command{
			{
				Name:   "create",
				Usage:  "Create a GitHub Discussion, via a Discussion category form",
				Action: handleCreate,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "repo",
						Usage: "Specify a repository. If omitted, uses current repository",
					},
					&cli.BoolFlag{
						Name:  "dry-run",
						Value: false,
						Usage: "Whether to take user input, but not submit via the API",
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error()+"\n")
		os.Exit(1)
	}
}

func handleCreate(c *cli.Context) error {
	repoOverride := c.String("repo")
	isDryRun := c.Bool("dry-run")

	if isDryRun {
		fmt.Printf("🧪🧪🧪🧪🧪🧪🧪🧪\n")
		fmt.Println("Running in dry run mode - will not create the discussion via the API")
		fmt.Println("To actually submit to the API, use `-dry-run=false`")
		fmt.Printf("🧪🧪🧪🧪🧪🧪🧪🧪\n")
	}

	var repo repository.Repository
	var err error

	if repoOverride == "" {
		repo, err = repository.Current()
	} else {
		repo, err = repository.Parse(repoOverride)
	}
	if err != nil {
		return fmt.Errorf("could not determine what repo to use: %w", err)
	}

	if repo.Host != "github.com" {
		// TODO: implement GitHub Enterprise
		return fmt.Errorf("using this extension with GitHub Enterprise is not supported at this time")
	}

	if repo.Owner == "" || repo.Name == "" {
		return fmt.Errorf("failed to determine a repository, found: %v/%v", repo.Owner, repo.Name)
	}

	fmt.Printf("Looking for Discussion category forms in %s/%s\n", repo.Owner, repo.Name)

	gqlClient, err := api.DefaultGraphQLClient()
	if err != nil {
		return fmt.Errorf("failed to construct GraphQL client: %w", err)
	}

	httpClient, err := api.DefaultHTTPClient()
	if err != nil {
		return fmt.Errorf("failed to construct HTTP client: %w", err)
	}

	dt := DiscussionTemplateClient{
		httpClient: httpClient,
		gqlClient:  gqlClient,
		repo:       repo,
	}

	categories, repositoryID, err := dt.Discover(c.Context)
	if err != nil {
		return fmt.Errorf("failed to discover repository's %v/%v's Discussion category forms: %w", repo.Owner, repo.Name, err)
	}

	if len(categories) == 0 {
		return fmt.Errorf("repository %v/%v did not have any Discussion category forms, exiting", repo.Owner, repo.Name)
	}

	prompt, err := CategoriesToPrompt(categories)
	if err != nil {
		return fmt.Errorf("failed to convert discovered category forms to an interactive prompt: %w", err)
	}

	var discussionSlug string
	err = survey.AskOne(prompt, &discussionSlug)
	if err != nil {
		return fmt.Errorf("failed to receive user input for category forms to use: %w", err)
	}

	categoryID := ""
	for _, category := range categories {
		if category.Slug == discussionSlug {
			categoryID = category.ID
			break
		}
	}

	if categoryID == "" {
		return fmt.Errorf("failed to discover the Discussion category ID for the slug %v: looked at %d elements, and found none that matched", discussionSlug, len(categories))
	}

	tpl, err := dt.RetrieveTemplate(c.Context, discussionSlug)
	if err != nil {
		return fmt.Errorf("failed to retrieve Discussion category form for slug %v: %w", discussionSlug, err)
	}

	var discussionTitle string
	q := survey.Input{
		Message: "Discussion title",
	}

	err = survey.AskOne(&q, &discussionTitle, survey.WithValidator(survey.Required))
	if err != nil {
		return err
	}

	var discussionBody string
	for i, item := range tpl.Body {
		q, label, askOpts, err := BodyItemToPromptAndOpts(item)
		if err != nil {
			return fmt.Errorf("failed to convert %dth element of type %v to an interactive prompt: %w", i, item.Type, err)
		}

		var result string
		err = survey.AskOne(q, &result, askOpts...)
		if err != nil {
			return err
		}

		discussionBody += "### " + label + "\n\n" + result + "\n\n"
	}

	if isDryRun {
		fmt.Printf("🧪🧪🧪🧪🧪🧪🧪🧪\n")
		fmt.Println("Running in dry run mode - will not create the discussion via the API")
		fmt.Printf("Would attempt to create a discussion with repositoryID=%#v categoryID=%#v len(body)=%d title=%#v with body:\n", repositoryID, categoryID, len(discussionBody), discussionTitle)
		fmt.Printf("----------------------------------------\n%s\n----------------------------------------\n", discussionBody)
		fmt.Println("To actually submit to the API, use `-dry-run=false`")
		fmt.Printf("🧪🧪🧪🧪🧪🧪🧪🧪\n")
	} else {
		url, err := dt.CreateDiscussion(c.Context, repositoryID, categoryID, discussionBody, discussionTitle)
		if err != nil {
			return fmt.Errorf("failed to create discussion: %w", err)
		}

		fmt.Printf("Successfully created %s\n", url)
	}

	return nil
}

type DiscussionTemplateClient struct {
	httpClient *http.Client
	gqlClient  *api.GraphQLClient
	repo       repository.Repository
}

type discoverQueryResponse struct {
	Repository struct {
		ID                   string
		DiscussionCategories struct {
			Edges []struct {
				Node struct {
					Name        string
					Description string
					ID          string
					Slug        string
				}
			}
		}
	}
}

const discoverQuery = `query ($owner: String!, $repo: String!) {
  repository(owner: $owner, name: $repo) {
    id
    discussionCategories(first: 25) {
      edges {
        node {
          name
          description
          id
          slug
        }
      }
    }
  }
}`

func (dt *DiscussionTemplateClient) Discover(ctx context.Context) ([]discussion.Category, string, error) {
	variables := map[string]any{
		"owner": dt.repo.Owner,
		"repo":  dt.repo.Name,
	}

	var resp discoverQueryResponse

	err := dt.gqlClient.DoWithContext(ctx, discoverQuery, variables, &resp)
	if err != nil {
		return nil, "", fmt.Errorf("failed to query repository %v/%v: %w", dt.repo.Owner, dt.repo.Name, err)
	}

	if len(resp.Repository.DiscussionCategories.Edges) == 0 {
		// NOTE that the caller should validate that this is empty
		return nil, "", nil
	}

	var categories []discussion.Category
	for _, category := range resp.Repository.DiscussionCategories.Edges {
		categories = append(categories, discussion.Category{
			ID:          category.Node.ID,
			Name:        category.Node.Name,
			Description: category.Node.Description,
			Slug:        category.Node.Slug,
		})
	}

	slices.SortFunc(categories, func(a, b discussion.Category) int {
		return strings.Compare(a.Slug, b.Slug)
	})

	return categories, resp.Repository.ID, nil
}

var errDiscussionTemplateNotFound = errors.New("no Discussion template could be found")

func (dt *DiscussionTemplateClient) RetrieveTemplate(ctx context.Context, slug string) (discussionform.Template, error) {
	gClient := github.NewClient(dt.httpClient)

	path := ".github/DISCUSSION_TEMPLATE/" + slug + ".yml"

	f, _, resp, err := gClient.Repositories.GetContents(ctx, dt.repo.Owner, dt.repo.Name, path, nil)
	if resp.StatusCode == http.StatusNotFound {
		return discussionform.Template{}, fmt.Errorf("no template could be found at path %s for repository %v/%v: %w", path, dt.repo.Owner, dt.repo.Name, errDiscussionTemplateNotFound)
	} else if err != nil {
		return discussionform.Template{}, fmt.Errorf("failed to look up template at path %s for repository %v/%v: %w", path, dt.repo.Owner, dt.repo.Name, err)
	}

	body, err := f.GetContent()
	if err != nil {
		return discussionform.Template{}, fmt.Errorf("failed to parse contents of file of length %d at path %s in repository %v/%v: %w", f.GetSize(), path, dt.repo.Owner, dt.repo.Name, err)
	}

	var tpl discussionform.Template
	if err := yaml.Unmarshal([]byte(body), &tpl); err != nil {
		return discussionform.Template{}, fmt.Errorf("failed to unmarshal contents of file of length %d at path %s in repository %v/%v as YAML: %w", f.GetSize(), path, dt.repo.Owner, dt.repo.Name, err)
	}

	return tpl, nil
}

const createDiscussionMutation = `
mutation CreateDiscussion(
  $repositoryId: ID!
  $categoryId: ID!
  $body: String!
  $title: String!
) {
  createDiscussion(
    input: {
      repositoryId: $repositoryId
      categoryId: $categoryId
      body: $body
      title: $title
    }
  ) {
    discussion {
      url
    }
  }
}
`

type createDiscussionResponse struct {
	CreateDiscussion struct {
		Discussion struct {
			URL string
		}
	}
}

func (dt *DiscussionTemplateClient) CreateDiscussion(ctx context.Context, repositoryID string, categoryID string, discussionBody string, discussionTitle string) (string, error) {
	variables := map[string]any{
		"repositoryId": repositoryID,
		"categoryId":   categoryID,
		"body":         discussionBody,
		"title":        discussionTitle,
	}

	var resp createDiscussionResponse

	err := dt.gqlClient.DoWithContext(ctx, createDiscussionMutation, variables, &resp)
	if err != nil {
		return "", fmt.Errorf("failed to create Discussion for repository %v/%v: %w", dt.repo.Owner, dt.repo.Name, err)
	}

	if resp.CreateDiscussion.Discussion.URL == "" {
		return "", fmt.Errorf("no URL was returned for the created Discussion for repository %v/%v", dt.repo.Owner, dt.repo.Name)
	}

	return resp.CreateDiscussion.Discussion.URL, nil
}

func CategoriesToPrompt(categories []discussion.Category) (survey.Prompt, error) {
	slugToPretty := make(map[string]string)

	for _, category := range categories {
		slugToPretty[category.Slug] = category.Description
	}

	options := make([]string, 0, len(slugToPretty))
	for k := range slugToPretty {
		options = append(options, k)
	}
	slices.Sort(options)

	return &survey.Select{
		Message: "Select category for new Discussion", // TODO
		Options: options,
		Description: func(value string, _ int) string {
			return slugToPretty[value]
		},
	}, nil
}

func BodyItemToPromptAndOpts(item discussionform.BodyItem) (survey.Prompt, string, []survey.AskOpt, error) {
	var label string

	ensureRequired := func(item discussionform.BodyItem) []survey.AskOpt {
		var askOpts []survey.AskOpt
		if isRequired, ok := item.Validations["required"]; ok && isRequired {
			askOpts = append(askOpts, survey.WithValidator(survey.Required))
		}
		return askOpts
	}

	switch t := item.Item.(type) {
	case discussionform.Dropdown:
		label = t.Attributes.Label
		askOpts := ensureRequired(item)

		if !slices.Contains(t.Attributes.Options, "None") {
			t.Attributes.Options = append(t.Attributes.Options, "None")
		}

		return &survey.Select{
			Message: t.Attributes.Label,
			Options: t.Attributes.Options,
			Default: "None",
		}, label, askOpts, nil

	case discussionform.Input:
		label = t.Attributes.Label
		askOpts := ensureRequired(item)

		return &survey.Input{
			Message: t.Attributes.Label,
		}, label, askOpts, nil

	case discussionform.Textarea:
		label = t.Attributes.Label
		askOpts := ensureRequired(item)

		pr := survey.Editor{
			Message:  t.Attributes.Label,
			Help:     t.Attributes.Description,
			FileName: "*.md",
		}

		if t.Attributes.Value != "" {
			pr.Default = t.Attributes.Value
			pr.AppendDefault = true
			pr.HideDefault = true
		}

		return &pr, label, askOpts, nil

	default:
		return nil, "", nil, fmt.Errorf("unexpected `type: %v` provided to BodyItemToQuestion", item.Type)
	}
}
