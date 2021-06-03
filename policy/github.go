package policy

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"text/template"

	"github.com/google/go-github/v35/github"
	"github.com/rs/zerolog/log"
)

const ghOrg = "TykTechnologies"

//go:embed pr-templates
var ghTemplates embed.FS

// (rp RepoPolicies) IsProtected tells you if a branch can be pushed directly to origin or needs to go via a PR
func (r *GitRepo) IsProtected(branch string) (bool, error) {
	b, resp, err := r.gh.Repositories.GetBranch(context.Background(), ghOrg, r.Name, branch)
	if err != nil {
		return true, fmt.Errorf("error: %w, response: %v", err, resp)
	}

	return b.GetProtected(), nil
}

func (r *GitRepo) CreatePR(title, templateName string, rp RepoPolicies, removal bool) error {
	if r.branch == "" {
		return fmt.Errorf("unknown local branch on repo %s when creating PR", r.Name)
	}
	if r.remoteBranch == "" {
		return fmt.Errorf("unknown remote branch on repo %s when creating PR", r.Name)
	}
	t := template.Must(template.
		New(templateName).
		Option("missingkey=error").
		ParseFS(ghTemplates, fmt.Sprintf("pr-templates/%s", templateName)))
	tv, err := rp.getPRVars(r.Name, r.branch, removal)
	if err != nil {
		return fmt.Errorf("template vars: %w", err)
	}
	log.Trace().Interface("tv", tv).Msg("template vars for PR")
	var b bytes.Buffer
	err = t.Execute(&b, tv)
	if err != nil {
		return fmt.Errorf("rendering template: %w", err)
	}
	prOpts := &github.NewPullRequest{
		Title: github.String(title),
		Head:  github.String(r.remoteBranch),
		Base:  github.String(r.branch),
		Body:  github.String(b.String()),
	}
	if r.dryRun {
		log.Warn().Msg("only dry-run, not really creating PR")
	} else {
		pr, _, err := r.gh.PullRequests.Create(context.Background(), ghOrg, r.Name, prOpts)
		if err != nil {
			return err
		}
		r.prs = append(r.prs, pr.GetHTMLURL())
	}
	return nil
}
