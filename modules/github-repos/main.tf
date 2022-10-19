terraform {

  required_providers {
    github = {
      source  = "integrations/github"
      version = "5.5.0"
    }
  }

}

resource "github_repository" "repository" {
  name                   = var.repo
  description            = var.description
  visibility             = "public"
  allow_rebase_merge     = false
  allow_squash_merge     = true
  allow_merge_commit     = false
  allow_auto_merge       = true
  delete_branch_on_merge = true
  has_downloads          = true
  has_issues             = true
  has_projects           = true
  topics                 = var.topics
}

resource "github_branch" "default" {
  repository = github_repository.repository.name
  branch     = var.default_branch
}

resource "github_branch_default" "default" {
  repository = github_repository.repository.name
  branch     = github_branch.default.branch
}

resource "github_branch_protection" "automerge" {
  repository_id = github_repository.repository.name
  pattern       = github_branch.default.branch

  #checks for automerge
  require_signed_commits          = true
  require_conversation_resolution = true

  required_status_checks {
    strict   = true
    contexts = var.required_status_checks_contexts
  }

  required_pull_request_reviews {
    require_code_owner_reviews      = true
    required_approving_review_count = var.required_approving_review_count
  }
}