package policy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"text/template"
	"time"

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
	Jira                 *JiraIssue
	BaseBranch, PrBranch string
	Owner, Repo          string
	AutoMerge            bool
}

var NoPRs = errors.New("no matching PRs found")

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

// CreatePR will create a PR using the user supplied title and the embedded PR body
// If a PR already exists, it will return that PR
func (gh *GithubClient) CreatePR(bv any, prOpts *PullRequest) (*github.PullRequest, error) {
	body, err := gh.RenderPRTemplate(&prOpts.Jira.Body, bv)
	if err != nil {
		return nil, err
	}
	title, err := gh.RenderPRTemplate(&prOpts.Jira.Title, bv)
	if err != nil {
		return nil, err
	}

	clientPROpts := &github.NewPullRequest{
		Title: github.String(fmt.Sprintf("[%s %s] %s", prOpts.Jira.Id, prOpts.BaseBranch, title.String())),
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
				log.Debug().Interface("ghErr", ghErr).Fields(resp).Msg("found existing PR")
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
			switch {
			case errors.Is(err, NoPRs):
				return nil, fmt.Errorf("possible bug in GithubClient.getPR()")
			default:
				return nil, fmt.Errorf("PR %s:%s exists but could not be fetched: %v", prOpts.Repo, prOpts.BaseBranch, err)
			}
		}
		pr.Title = clientPROpts.Title
		pr.Body = clientPROpts.Body
		prNum := pr.GetNumber()
		pr, resp, err := gh.v3.PullRequests.Edit(context.Background(), prOpts.Owner, prOpts.Repo, prNum, pr)
		log.Trace().Fields(resp).Str("owner", prOpts.Owner).Str("repo", prOpts.Repo).Msgf("updating PR#%d", prNum)
		if err != nil {
			return pr, fmt.Errorf("Updating PR#%d failed: %w", prNum, err)
		}
		log.Info().Msgf("Updated existing PR#%d", prNum)
	}
	log.Trace().Interface("pr", pr).Msgf("PR %s/%s<-%s", prOpts.Owner, prOpts.BaseBranch, prOpts.PrBranch)
	if prOpts.AutoMerge {
		err = gh.EnableAutoMerge(pr.GetNodeID())
	}
	return pr, err
}

// getPR searches for PRs created for the head ref/branch
func (gh *GithubClient) getPR(prOpts *PullRequest) (*github.PullRequest, error) {
	prlOpts := &github.PullRequestListOptions{
		Base: prOpts.BaseBranch,
		Head: prOpts.Owner + ":" + prOpts.PrBranch,
	}
	prs, resp, err := gh.v3.PullRequests.List(context.Background(), prOpts.Owner, prOpts.Repo, prlOpts)
	if err != nil {
		return nil, fmt.Errorf("listing PRs: %v", err)
	}
	log.Trace().Interface("resp", resp).Interface("prs", prs).Msg("getting existing PRs")
	if len(prs) > 0 {
		return prs[0], nil
	} else {
		return nil, NoPRs
	}
}

// (gh *GithubClient) ClosePR will close matching PRs without merging
func (gh *GithubClient) ClosePR(prOpts *PullRequest) error {
	pr, err := gh.getPR(prOpts)
	if err != nil {
		switch {
		case errors.Is(err, NoPRs):
			log.Info().Msgf("No releng PRs found for %s:%s<-%s", prOpts.Repo, prOpts.BaseBranch, prOpts.PrBranch)
			return nil
		default:
			return err
		}
	}
	pr.State = github.String("closed")
	pr, resp, err := gh.v3.PullRequests.Edit(context.Background(), prOpts.Owner, prOpts.Repo, *pr.Number, pr)
	log.Trace().Interface("resp", resp).Interface("pr", pr).Msg("closing PR")
	log.Info().Msgf("closed %s#%d", prOpts.Repo, *pr.Number)
	return err
}

// (gh *GithubClient) UpdatePR will update prOpts.PrBranch without needing a git checkout
func (gh *GithubClient) UpdatePrBranch(prOpts *PullRequest) error {
	pr, err := gh.getPR(prOpts)
	if err != nil {
		switch {
		case errors.Is(err, NoPRs):
			log.Info().Msgf("No releng PRs found for %s:%s<-%s", prOpts.Repo, prOpts.BaseBranch, prOpts.PrBranch)
			return nil
		default:
			return err
		}
	}
	attempts := 3
	delay := time.Second * 2
again:
	// Default value of pruOpts should DTRT
	var pruOpts github.PullRequestBranchUpdateOptions
	pru, resp, err := gh.v3.PullRequests.UpdateBranch(context.Background(), prOpts.Owner, prOpts.Repo, *pr.Number, &pruOpts)
	log.Trace().Interface("resp", resp).Interface("pr", pru).Msgf("updating branch for %s:%s<-%s", prOpts.Repo, prOpts.BaseBranch, prOpts.PrBranch)
	_, isae := err.(*github.AcceptedError)
	if attempts > 0 && !isae {
		attempts--
		log.Debug().Msgf("Waiting %s to try again", delay)
		time.Sleep(delay)
		goto again
	}
	return err
}

// (gh *GithubClient) Open will open the PR matching prOpts in the default browser
func (gh *GithubClient) Open(prOpts *PullRequest) error {
	pr, err := gh.getPR(prOpts)
	if err == nil {
		return openInBrowser(*pr.HTMLURL)
	}
	return err
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

// https://stackoverflow.com/questions/39320371/how-start-web-server-to-open-page-in-browser-in-golang
// open opens the specified URL in the default browser of the user.
func openInBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}
