package policy

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJiraGetIssue(t *testing.T) {
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
	} else {
		t.Fatalf("Could not get simple issue SYSE-2: %v", err)
	}
}

func TestJiraGetEpic(t *testing.T) {
	user := os.Getenv("JIRA_USER")
	token := os.Getenv("JIRA_TOKEN")
	if token == "" || user == "" {
		t.Skip("Requires JIRA_USER and JIRA_TOKEN be set to run this test.")
	}
	j := NewJiraClient(user, token)
	i, err := j.GetIssue("SYSE-358")
	if assert.NoError(t, err) {
		assert.Equal(t, "SYSE-358", i.Id)
		assert.Equal(t, `Skip signing when building a snapshot to allow dependabot PRs to build.
Use larger runners for build and test.
- [x] runner m2 amends
- [x] Prepare for release-5.3 becoming LTS
- [x] Start collecting new reports for UI and API tests
- [x] Add r4-lts to Dr. Releng
`, i.Body)
	} else {
		t.Fatalf("Could not get epic SYSE-358: %v", err)
	}
}
