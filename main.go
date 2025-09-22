package main

import (
	"fmt"
	"os"

	"github.com/cli/go-gh/v2/pkg/api"
)

func main() {
	if err := run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "gh-discussion failed: %s\n", err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	fmt.Println("hi world, this is the gh-discussion extension!")
	client, err := api.DefaultRESTClient()
	if err != nil {
		return err
	}
	response := struct{ Login string }{}
	err = client.Get("user", &response)
	if err != nil {
		return err
	}
	fmt.Printf("running as %s\n", response.Login)

	return nil
}

// For more examples of using go-gh, see:
// https://github.com/cli/go-gh/blob/trunk/example_gh_test.go
