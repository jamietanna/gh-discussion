package main

import (
	"context"
	"flag"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"slices"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/jamietanna/gh-discussion/internal/discussion"
	"github.com/jamietanna/gh-discussion/internal/discussionform"
)

func main() {
	if err := run(os.Args, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "gh-discussion failed: %s\n", err.Error())
		os.Exit(1)
	}
}

func run(args []string, stdin *os.File, stdout *os.File, stderr *os.File) error {
	_ = context.Background()

	repoOverride := flag.String(
		"repo", "", "Specify a repository. If omitted, uses current repository")
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

	fmt.Printf(
		"Going to search discussions in %s/%s\n", repo.Owner, repo.Name)

	// client, err := api.DefaultHTTPClient()
	// if err != nil {
	// 	return fmt.Errorf("failed to construct REST client: %w", err)
	// }
	//
	// gClient := github.NewClient(client)
	//
	// _, d, resp, err := gClient.Repositories.GetContents(ctx, repo.Owner, repo.Name, ".github/DISCUSSION_TEMPLATE", nil)
	// if resp.StatusCode == http.StatusNotFound {
	// 	// TODO no template
	// 	return fmt.Errorf("TODO: no template")
	// } else if err != nil {
	// 	return fmt.Errorf("failed to TODO: %w", err)
	// }
	//
	// fmt.Printf("d: %v\n", d)
	//
	// for _, file := range d {
	// 	// file.
	// }
	//
	//
	// categories := []struct{
	// 	Name string
	// 	Description string
	// 	Slug string
	// }

	dt := DiscussionTemplate_{}

	categories, err := dt.Discover()
	if err != nil {
		return fmt.Errorf("TODO: %w", err)
	}

	slices.SortFunc(categories, func(a, b discussion.Category) int {
		return strings.Compare(a.Slug, b.Slug)
	})

	var options []string
	for _, c := range categories {
		options = append(options, c.Name+" ("+c.Description+")")
	}

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

	// HACK
	data, err := os.ReadFile("../renovate/.worktrees/HEAD/.github/DISCUSSION_TEMPLATE/request-help.yml")
	if err != nil {
		return err
	}

	var tpl discussionform.Template
	if err := yaml.Unmarshal(data, &tpl); err != nil {
		return err // TODO
	}

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

		q, askOpts, err := BodyItemToQuestion(item)
		if err != nil {
			return err
		}

		m := make(map[string]any)
		err = survey.AskOne(q, &m, askOpts...)
		if err != nil {
			return err
		}
	}

	// HACK

	return nil
}

type DiscussionTemplate_ struct{}

func (dt *DiscussionTemplate_) Discover() ([]discussion.Category, error) {
	return []discussion.Category{
		{
			Name:        "Request Help",
			Description: "Ask here for help getting your config right or if you think you've found a bug",
			Slug:        "request-help",
		},
		{
			Name:        "Suggest an Idea",
			Description: "Start here if you have a feature request or idea for Renovate",
			Slug:        "suggest-an-idea",
		},
	}, nil
}

func (dt *DiscussionTemplate_) Get___(slug string) []any {
	return nil
}

// For more examples of using go-gh, see:
// https://github.com/cli/go-gh/blob/trunk/example_gh_test.go

func BodyItemToQuestion(item discussionform.BodyItem) (survey.Prompt, []survey.AskOpt, error) {
	var askOpts []survey.AskOpt

	ensureRequired := func(item discussionform.BodyItem) {
		fmt.Printf("item.Validations: %v\n", item.Validations)
		if isRequired, ok := item.Validations["required"]; ok && isRequired {
			askOpts = append(askOpts, survey.WithValidator(survey.Required))
		}
	}

	switch t := item.Item.(type) {
	case discussionform.Dropdown:
		ensureRequired(item)

		if !slices.Contains(t.Attributes.Options, "None") {
			t.Attributes.Options = append(t.Attributes.Options, "None")
		}

		return &survey.Select{
			Message: t.Attributes.Label,
			Options: t.Attributes.Options,
			Default: "None",
		}, askOpts, nil

	case discussionform.Input:
		ensureRequired(item)

		return &survey.Input{
			Message: t.Attributes.Label,
		}, askOpts, nil

	case discussionform.Textarea:
		ensureRequired(item)

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

		// Example: set an AskOpt for Textarea

		return &pr, askOpts, nil

	default:
		return nil, nil, fmt.Errorf("unexpected `type: %v` provided to BodyItemToQuestion", item.Type)
	}
}
