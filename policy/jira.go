package policy

import (
	"context"
	"fmt"
	"strings"

	v3 "github.com/ctreminiom/go-atlassian/jira/v3"
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
	i, resp, err := j.c.Issue.Get(j.ctx, id, []string{"summary", "description", "subtasks"}, nil)
	log.Trace().Interface("resp", resp).Interface("issue", i).Msg("getissue response")
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
	for _, st := range i.Fields.Subtasks {
		b += fmt.Sprintf("- [x] %s\n", st.Fields.Summary)
	}
	return &JiraIssue{
		Id:    i.Key,
		Title: i.Fields.Summary,
		Body:  b,
	}, err
}
