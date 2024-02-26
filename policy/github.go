package policy

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	_ "embed"

	"github.com/Masterminds/sprig/v3"
	"github.com/google/go-github/v59/github"
	"github.com/rs/zerolog/log"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type GithubClient struct {
	v3 *github.Client
	v4 *githubv4.Client
}

type PullRequest struct {
	Title                string
	BaseBranch, PrBranch string
	Owner, Repo          string
}

// NewGithubClient returns a client that uses the v3 (REST) API to talk to Github
func NewGithubClient(ghToken string) *GithubClient {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghToken},
	)
	tc := oauth2.NewClient(context.Background(), ts)

	return &GithubClient{
		v3: github.NewClient(tc),
		v4: githubv4.NewClient(tc),
	}
}

// RenderPRTemplate will fill in the supplied template body with values from GitRepo
func (gh *GithubClient) RenderPRTemplate(body *string, bv any) (*bytes.Buffer, error) {
	op := new(bytes.Buffer)
	t := template.Must(
		template.New("prbody").
			Option("missingkey=error").
			Funcs(sprig.FuncMap()).
			Parse(*body))
	err := t.Execute(op, bv)
	return op, err
}

//go:embed prs/main.tmpl
var prbody string

// CreatePR will create a PR using the user supplied title and the embedded PR body
// If a PR already exists, it will return that PR
func (gh *GithubClient) CreatePR(bv any, prOpts *PullRequest) (*github.PullRequest, error) {
	body, err := gh.RenderPRTemplate(&prbody, bv)
	if err != nil {
		return nil, err
	}
	title, err := gh.RenderPRTemplate(&prOpts.Title, bv)
	if err != nil {
		return nil, err
	}

	clientPROpts := &github.NewPullRequest{
		Title: github.String(title.String()),
		Head:  github.String(prOpts.PrBranch),
		Base:  github.String(prOpts.BaseBranch),
		Body:  github.String(body.String()),
		Draft: github.Bool(false),
	}
	log.Trace().Interface("propts", prOpts).Str("owner", prOpts.Owner).Str("repo", prOpts.Repo).Msg("creating PR")
	pr, resp, err := gh.v3.PullRequests.Create(context.Background(), prOpts.Owner, prOpts.Repo, clientPROpts)
	// Attempt to detect if a PR already existingPR, complexity due to
	// https://github.com/google/go-github/issues/1441
	existingPR := false
	if e, ok := err.(*github.ErrorResponse); ok {
		for _, ghErr := range e.Errors {
			if strings.HasPrefix(ghErr.Message, "A pull request already exists") {
				log.Debug().Interface("ghErr", ghErr).Interface("resp", resp).Msg("found existing PR")
				existingPR = true
				break
			}
		}
	}
	if !existingPR && err != nil {
		return nil, fmt.Errorf("error creating PR for %s:%s: %v", prOpts.Repo, prOpts.BaseBranch, err)
	} else if existingPR {
		pr, err = gh.getPR(prOpts)
		if err != nil {
			return nil, fmt.Errorf("PR %s:%s exists but could not be fetched: %v", prOpts.Repo, prOpts.BaseBranch, err)
		}
	}
	log.Trace().Interface("pr", pr).Msgf("PR %s/%s<-%s", prOpts.Owner, prOpts.BaseBranch, prOpts.PrBranch)
	return pr, nil
}

// getPR searches for PRs created for the head ref/branch
func (gh *GithubClient) getPR(prOpts *PullRequest) (*github.PullRequest, error) {
	prlOpts := &github.PullRequestListOptions{
		Head: prOpts.BaseBranch,
	}
	prs, resp, err := gh.v3.PullRequests.List(context.Background(), prOpts.Owner, prOpts.Repo, prlOpts)
	log.Trace().Interface("resp", resp).Msg("getting existing PR")
	for _, pr := range prs {
		if prOpts.BaseBranch == pr.Base.GetRef() {
			return pr, err
		}
	}
	return nil, err
}

// EnableAutoMergePR uses the graphQL github v4 API with the PR ID
// (not number) to mutate graphQL PR object to enable automerge
func (gh *GithubClient) EnableAutoMerge(prID string) error {
	var mutation struct {
		Automerge struct {
			ClientMutationID githubv4.String
			Actor            struct {
				Login githubv4.String
			}
			PullRequest struct {
				BaseRefName githubv4.String
				CreatedAt   githubv4.DateTime
				Number      githubv4.Int
			}
		} `graphql:"enablePullRequestAutoMerge(input: $input)"`
	}

	mergeMethod := githubv4.PullRequestMergeMethodSquash

	amInput := githubv4.EnablePullRequestAutoMergeInput{
		MergeMethod:   &mergeMethod,
		PullRequestID: prID,
	}

	return gh.v4.Mutate(context.Background(), &mutation, amInput, nil)
}
