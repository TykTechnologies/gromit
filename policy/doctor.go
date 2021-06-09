package policy

import (
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/rs/zerolog/log"
	"embed"
)

const saPath = ".github/workflows/sync-automation.yml"

// CheckMetaAutomation checks if sync-automation is present in this repo
func (r *GitRepo) CheckMetaAutomation(rp RepoPolicies) error {
	srcBranches, err := rp.SrcBranches(r.Name)
	if err != nil {
		return fmt.Errorf("fetch src branches: %w", err)
	}

	shouldExist := false
	for _, branch := range srcBranches {
		if branch == r.branch {
			shouldExist = true
			break
		}
	}
	exists := false
	_, err = r.fs.Stat(saPath)
	if err == nil {
		exists = true
	}

	var prTitle string
	var isRemoval bool
	if exists == shouldExist {
		return nil
	} else if shouldExist == true {
		err = r.AddMetaAutomation("releng: :syringe: the doctor re-attached your meta-automation :muscle:", rp)
		if err != nil {
			return fmt.Errorf("adding meta automation: %w", err)
		}
		prTitle = fmt.Sprintf("releng: automatically port commits from branch %s on %s", r.branch, r.Name)
		isRemoval = false
	} else if shouldExist == false {
		err = r.RemoveMetaAutomation("releng: :space_invader: the doctor had to amputate your meta-automation :hocho:")
		if err != nil {
			return fmt.Errorf("removing meta automation: %w", err)
		}
		prTitle = fmt.Sprintf("releng: removing meta automation from branch %s on %s", r.branch, r.Name)
		isRemoval = true
	} else {
		// file doesn't exist and shouldn't exist
		return nil
	}
	log.Debug().Bool("exists", exists).Bool("shouldExist", shouldExist).Msg("current vs desired")
	var remoteBranch string
	isProtected, err := r.IsProtected(r.branch)
	if err != nil {
		return fmt.Errorf("getting protected status: %w", err)
	}
	if isProtected {
		log.Info().Bool("isProtected", isProtected).Msg("will create PR instead of pushing upstream")
		remoteBranch = fmt.Sprintf("releng/%s", r.branch)
		err = r.Push(r.branch, remoteBranch)
		if err != nil {
			return fmt.Errorf("pushing to %s: %w", defaultRemote, err)
		}
		err = r.CreatePR(prTitle, "sync-automation.tmpl", rp, isRemoval)
		if err != nil {
			return fmt.Errorf("creating PR: %w", err)
		}
		log.Info().Msg("created PR")
	} else {
		log.Info().Bool("isProtected", isProtected).Msg("pushing directly to upstream")
		remoteBranch = r.branch
		err = r.Push(r.branch, remoteBranch)
		if err != nil {
			return fmt.Errorf("pushing to %s: %w", defaultRemote, err)
		}
	}
	log.Info().Str("origin", remoteBranch).Msg("branch pushed to origin")

	return nil
}

// (r *GitRepo) RemoveMetaAutomation removes sync-automation from the repo for the given branch
// If protected==true the change is not pushed directly upstream but to a releng/ branch
func (r *GitRepo) RemoveMetaAutomation(commitMsg string) error {
	hash, err := r.worktree.Remove(saPath)
	if err != nil {
		return fmt.Errorf("removing: %w", err)
	}
	log.Trace().Str("hash", hash.String()).Msg("remove from worktree")
	newCommitHash, err := r.worktree.Commit(commitMsg, r.commitOpts)
	if err != nil {
		return fmt.Errorf("committing to worktree: %w", err)
	}
	log.Trace().Str("hash", newCommitHash.String()).Msg("committing removal")
	return err
}

//go:embed action-templates
var maTemplates embed.FS

// AddMetaAutomation generates the backport meta-automation .g/w/sync-automation.yml
func (r *GitRepo) AddMetaAutomation(commitMsg string, rp RepoPolicies) error {
	log.Info().Msg("generating sync-automation.yml")

	opFile := filepath.Join(".github", "workflows", "sync-automation.yml")
	op, err := r.CreateFile(opFile)
	if err != nil {
		return err
	}
	defer op.Close()

	t := template.Must(template.
		New("sync-automation.tmpl").
		Option("missingkey=error").
		ParseFS(maTemplates, "action-templates/sync-automation.tmpl"))
	if err != nil {
		return fmt.Errorf("template vars: %w", err)
	}
	log.Debug().Msg("writing .g/w/sync-automation.yml")
	tv, err := rp.getMAVars(r.Name, r.branch)
	if err != nil {
		return fmt.Errorf("getting tvars for %s/%s: %w", r.Name, r.branch, err)
	}
	log.Trace().Interface("tv", tv).Msg("template vars for adding meta automation")
	err = t.Execute(op, tv)
	if err != nil {
		return fmt.Errorf("rendering template: %w", err)
	}
	_, err = r.addFile(opFile, commitMsg, true)
	return err
}
