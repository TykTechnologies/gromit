package policy

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetIssue(t *testing.T) {
	user := os.Getenv("JIRA_USER")
	token := os.Getenv("JIRA_TOKEN")
	if token == "" || user == "" {
		t.Skip("Requires JIRA_USER and JIRA_TOKEN be set to run this test.")
	}
	j := NewJiraClient(user, token)
	i, err := j.GetIssue("SYSE-2")
	if assert.NoError(t, err) {
		assert.Equal(t, "SYSE-2", i.Id)
		assert.Equal(t, "Authentication for internal tools", i.Title)
	}
}
