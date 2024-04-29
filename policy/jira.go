package policy

import (
	"context"
	"fmt"
	"strings"

	v3 "github.com/ctreminiom/go-atlassian/jira/v3"
	"github.com/ctreminiom/go-atlassian/pkg/infra/models"
	"github.com/rs/zerolog/log"
)

type JiraClient struct {
	c   *v3.Client
	ctx context.Context
}

type JiraIssue struct {
	Id    string
	Title string
	Body  string
}

// NewJiraClient returns a client for v3 REST operations
func NewJiraClient(email, token string) *JiraClient {
	c, err := v3.New(nil, "https://tyktech.atlassian.net")
	if err != nil {
		log.Fatal().Err(err).Msg("Getting Jira v3 client")
	}
	c.Auth.SetBasicAuth(email, token)
	return &JiraClient{
		c: c,
	}
}

// (j *JiraClient) GetIssue returns the issue after serialising the description
// Jira v3 API returns a structured version of the description, this function only
// understands a few types. Unknown content types are ignored.
func (j *JiraClient) GetIssue(id string) (*JiraIssue, error) {
	j.ctx = context.Background()
	i, resp, err := j.c.Issue.Get(j.ctx, id, []string{"summary", "description", "subtasks", "issuetype"}, nil)
	log.Trace().Fields(resp).Interface("issue", i).Msg("getissue response")
	if err != nil {
		return nil, err
	}
	var b string
	for _, c := range i.Fields.Description.Content {
		switch c.Type {
		case "paragraph":
			for _, p := range c.Content {
				if p.Type == "text" {
					b += p.Text
				}
			}
			b += "\n"
		case "heading":
			b += strings.Repeat("#", c.Attrs["level"].(int))
			for _, cc := range c.Content {
				if cc.Type == "text" {
					b += cc.Text
				}
			}
			b += "\n"
		default:
		}
	}
	b += getChildLines(i.Fields.Subtasks)
	if i.Fields.IssueType.Name == "Epic" {
		jql := fmt.Sprintf("parent = %s", id)
		cis, resp, err := j.c.Issue.Search.Get(j.ctx, jql, nil, nil, 0, 20, "stories")
		log.Trace().Fields(resp).Interface("issue", i).Msg("searching for children")
		if err != nil {
			log.Error().Err(err).Msgf("error fetching children of %s", id)
		} else {
			log.Debug().Msgf("found %d children", cis.Total)
			b += getChildLines(cis.Issues)
		}
	}
	return &JiraIssue{
		Id:    i.Key,
		Title: i.Fields.Summary,
		Body:  b,
	}, err
}

// getChildLines(parent) returns lines of the form
// - [ ] summary text
// with the checkbox filled in if the task/story is done
func getChildLines(parent []*models.IssueScheme) string {
	var b string
	for _, child := range parent {
		var status string
		if child.Fields.Status.StatusCategory.Name == "Done" {
			status = "x"
		} else {
			status = " "
		}
		b += fmt.Sprintf("- [%s] %s\n", status, child.Fields.Summary)
	}
	return b
}
