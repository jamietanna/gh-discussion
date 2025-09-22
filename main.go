package main

import (
	"context"
	"flag"
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
)

func main() {
	ctx := context.Background()

	if err := run(ctx, os.Args, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "gh-discussion failed: %s\n", err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdin *os.File, stdout *os.File, stderr *os.File) error {
	repoOverride := flag.String("repo", "", "Specify a repository. If omitted, uses current repository")
	isDryRun := flag.Bool("dry-run", true, "Whether to simulate **??**, but not submit **??**") // TODO
	flag.Parse()

	var repo repository.Repository
	var err error

	if repoOverride == nil || *repoOverride == "" {
		repo, err = repository.Current()
	} else {
		repo, err = repository.Parse(*repoOverride)
	}
	if err != nil {
		return fmt.Errorf("could not determine what repo to use: %w", err)
	}

	if repo.Owner == "" || repo.Name == "" {
		return fmt.Errorf("failed to determine a repository, found: %v/%v/%v", repo.Host, repo.Owner, repo.Name)
	}

	fmt.Printf("Looking for discussion category forms in %s/%s\n", repo.Owner, repo.Name)

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

	categories, repositoryID, err := dt.Discover(ctx)
	if err != nil {
		return fmt.Errorf("TODO: %w", err)
	}

	fmt.Printf("categories: %v\n", categories)

	fmt.Printf("repositoryID: %v\n", repositoryID) // TODO

	prompt, err := CategoriesToPrompt(categories)
	if err != nil {
		return fmt.Errorf("TODO: %w", err)
	}

	var discussionSlug string
	err = survey.AskOne(prompt, &discussionSlug)
	if err != nil {
		return fmt.Errorf("TODO: %w", err)
	}

	fmt.Printf("discussionSlug: %v\n", discussionSlug)

	tpl, err := dt.RetrieveTemplate(ctx, discussionSlug)
	if err != nil {
		return fmt.Errorf("TODO: %w", err)
	}

	// // TODO wrap in function
	//
	// var options []string
	// for _, c := range categories {
	// 	options = append(options, c.Name+" ("+c.Description+")")
	// }
	//
	// templateSelect := survey.Select{
	// 	Message: "Choose TODO",
	// 	Options: options,
	// }
	//
	// var slug string
	//
	// TODO wrap in function

	// var options []string
	// for _, c := range categories {
	// 	options = append(options, c.Name+" ("+c.Description+")")
	// }
	//
	// 	prompt := survey.Editor{
	// 		Message: "Please tell us more about your question or problem",
	// 		Help: `Remember to [follow these guidelines](https://github.com/renovatebot/renovate/blob/main/docs/development/help-us-help-you.md) for maximum effectiveness.
	// It may help to include a [minimal reproduction](https://github.com/renovatebot/renovate/blob/main/docs/development/minimal-reproductions.md) as well.
	// 		`,
	// 		FileName: "*.md",
	// 	}
	//
	// 	m := make(map[string]any)
	//
	// 	var result string
	//
	// 	err = survey.AskOne(&prompt, &result)
	// 	fmt.Printf("err: %v\n", err)
	//
	// 	fmt.Printf("m: %v\n", m)
	//
	// fmt.Printf("result: %v\n", result)

	// pr := prompter.New(stdin, stdout, stderr)
	// i, err := pr.Select("Choose which discussion to create", "", options)
	// fmt.Printf("err: %v\n", err)
	// fmt.Printf("i: %v\n", i)
	//
	// i, err = pr.Select("How are you running Renovate?", "None", []string{"A Mend.io-hosted app", "Self-hosted Renovate", "None"})
	// fmt.Printf("err: %v\n", err)
	// fmt.Printf("i: %v\n", i)
	//
	// pr.Input

	// // HACK
	// data, err := os.ReadFile("../renovate/.worktrees/HEAD/.github/DISCUSSION_TEMPLATE/request-help.yml")
	// // data, err := os.ReadFile("../renovate/.worktrees/HEAD/.github/DISCUSSION_TEMPLATE/request-help.yml")
	// if err != nil {
	// 	return err
	// }
	//
	// var tpl discussionform.Template
	// if err := yaml.Unmarshal(data, &tpl); err != nil {
	// 	return err // TODO
	// }

	var discussionTitle string

	q := survey.Input{
		Message: "Discussion title",
	}

	err = survey.AskOne(&q, &discussionTitle, survey.WithValidator(survey.Required))
	if err != nil {
		return err
	}

	var discussionBody string

	for _, item := range tpl.Body {
		// switch item.Type {
		// case "textarea":
		// 	if textarea, ok := item.Item.(discussionform.Textarea); ok {
		// 		attrs := textarea.Attributes
		// 		fmt.Printf("attrs: %v\n", attrs)
		// 	}
		//
		// // case "input":
		// // item.Item.(
		// //
		// // prompt := survey.Input{
		// // }
		// default:
		// 	fmt.Printf("item.Type: %v\n", item.Type)
		// }

		q, label, askOpts, err := BodyItemToPromptAndOpts(item)
		if err != nil {
			return err
		}

		var result string
		err = survey.AskOne(q, &result, askOpts...)
		if err != nil {
			return err
		}

		fmt.Printf("result: %v\n", result)

		discussionBody += "### " + label + "\n\n" + result + "\n\n"
	}

	categoryID := "TODO"

	for _, category := range categories {
		if category.Slug == discussionSlug {
			categoryID = category.ID
			break
		}
	}
	if categoryID == "TODO" {
		return fmt.Errorf("TODO: NO CAT")
	}

	if isDryRun == nil || *isDryRun {
		fmt.Printf("🧪🧪🧪🧪🧪🧪🧪🧪\n")
		fmt.Printf("Would attempt to create a discussion with repositoryID=%v categoryID=%v len(body)=%d title=%v\n", repositoryID, categoryID, len(discussionBody), discussionTitle)
		fmt.Println("To actually **??**, use `-dry-run false`")

		fmt.Printf("%s\n", discussionBody)
	} else {
		url, err := dt.CreateDiscussion(ctx, repositoryID, categoryID, discussionBody, discussionTitle)

		if err != nil {
			return err
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

func (dt *DiscussionTemplateClient) Discover(ctx context.Context) ([]discussion.Category, string, error) {
	query := `query ($owner: String!, $repo: String!) {
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

	variables := map[string]any{
		"owner": dt.repo.Owner,
		"repo":  dt.repo.Name,
	}

	var resp discoverQueryResponse

	err := dt.gqlClient.DoWithContext(ctx, query, variables, &resp)
	if err != nil {
		return nil, "", fmt.Errorf("TODO: %w", err)
	}

	fmt.Printf("resp: %v\n", resp)

	if len(resp.Repository.DiscussionCategories.Edges) == 0 {
		return nil, "", fmt.Errorf("TODO: None")
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

func (dt *DiscussionTemplateClient) RetrieveTemplate(ctx context.Context, slug string) (discussionform.Template, error) {
	gClient := github.NewClient(dt.httpClient)

	f, _, resp, err := gClient.Repositories.GetContents(ctx, dt.repo.Owner, dt.repo.Name, ".github/DISCUSSION_TEMPLATE/"+slug+".yml", nil)
	if resp.StatusCode == http.StatusNotFound {
		// TODO no template
		return discussionform.Template{}, fmt.Errorf("TODO: no template")
	} else if err != nil {
		return discussionform.Template{}, fmt.Errorf("failed to TODO: %w", err)
	}

	body, err := f.GetContent()
	if err != nil {
		return discussionform.Template{}, fmt.Errorf("failed to TODO: %w", err)
	}

	var tpl discussionform.Template
	if err := yaml.Unmarshal([]byte(body), &tpl); err != nil {
		return discussionform.Template{}, err // TODO
	}

	return tpl, nil
}

type createDiscussionResponse struct {
	CreateDiscussion struct {
		Discussion struct {
			URL string
		}
	}
}

func (dt *DiscussionTemplateClient) CreateDiscussion(ctx context.Context, repositoryID string, categoryID string, discussionBody string, discussionTitle string) (string, error) {
	query := `
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
	variables := map[string]any{
		"repositoryId": repositoryID,
		"categoryId":   categoryID,
		"body":         discussionBody,
		"title":        discussionTitle,
	}

	var resp createDiscussionResponse

	err := dt.gqlClient.DoWithContext(ctx, query, variables, &resp)
	if err != nil {
		return "", fmt.Errorf("TODO: %w", err)
	}

	fmt.Printf("resp: %v\n", resp)

	if resp.CreateDiscussion.Discussion.URL == "" {
		return "", fmt.Errorf("TODO empty")
	}

	// TODO: Implement the logic to create a discussion
	// This is a stub implementation
	// Replace with actual API call or logic as needed
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
		Message: "TODO", // TODO
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
