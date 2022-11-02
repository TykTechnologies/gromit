package main

import (
	"context"
	"fmt"
	"os"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

var query struct {
	Viewer struct {
		Login     githubv4.String
		CreatedAt githubv4.DateTime
	}
}

var mutation struct {
	Automerge struct {
		clientMutationId githubv4.String
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

func main() {

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	client := githubv4.NewClient(httpClient)
	// Use client...

	squash := githubv4.PullRequestMergeMethodSquash

	input := githubv4.EnablePullRequestAutoMergeInput{
		AuthorEmail:    githubv4.NewString("policy@gromit"),
		CommitBody:     githubv4.NewString("This PR will automerge. Enabled by Gromit"),
		CommitHeadline: githubv4.NewString("Automerge"),
		MergeMethod:    &squash,
		PullRequestID:  githubv4.NewBase64String("PR_kwDOASogO85AxbF1"),
	}

	err := client.Mutate(context.Background(), &mutation, input, nil)

	if err != nil {
		fmt.Println("Error: ", err)
	}

}
