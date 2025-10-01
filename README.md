# `gh-discussion`: a GitHub CLI extension to interact with GitHub Discussions

An extension for [the GitHub CLI](https://cli.github.com/) which allows interacting with the Discussions API.

## Features

- Create a GitHub Discussion from the `gh` CLI, using any [Discussion category forms](https://docs.github.com/en/discussions/managing-discussions-for-your-community/creating-discussion-category-forms)

### Caveats

There are a number of other known issues, gaps and features noted in the [this repo's Issues](https://github.com/jamietanna/gh-discussion).

A number of key callouts:

- This does not (yet) support GitHub Enterprise

## Installation

```sh
gh extension install jamietanna/gh-discussion
```

## Usage

Create a new discussion in the current repository:

```sh
gh discussion create
```

Specify a repository (`owner/repo`):

```sh
gh discussion create --repo owner/repo
```

Preview the discussion without submitting (`dry-run`, default):

```sh
gh discussion create --dry-run
```

Submit the discussion to GitHub:

```sh
gh discussion create --dry-run=false
```

## Flags

- `--repo`: Specify a repository (i.e. `renovatebot/renovate`). If omitted, uses current repository
- `--dry-run`: Whether to take user input, but not submit via the API. Default is `true`.

## License

This is licensed under the Apache-2.0.

Note that some of the code within `internal/discussionform/`, and this `README.md`, includes output from the following Large Language Models (LLMs), via GitHub Copilot:

- gpt-4.1
