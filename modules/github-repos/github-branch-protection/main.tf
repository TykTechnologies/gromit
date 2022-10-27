terraform {

  required_providers {
    github = {
      source  = "integrations/github"
      version = "5.5.0"
    }
  }

}

resource "github_branch_protection" "automerge" {
  repository_id = var.repo                           //github_repository.repository.name
  pattern       = var.branch_protection_conf.pattern //github_branch.default.branch

  #checks for automerge
  require_signed_commits          = var.branch_protection_conf.signed_commits // Lets discuss about this one before implement
  require_conversation_resolution = false
  required_linear_history         = var.branch_protection_conf.linear_history
  enforce_admins                  = false
  allows_deletions                = var.branch_protection_conf.allows_deletions
  allows_force_pushes             = var.branch_protection_conf.allows_force_pushes
  blocks_creations                = var.branch_protection_conf.blocks_creations
  push_restrictions               = var.branch_protection_conf.push_restrictions

  required_status_checks {
    strict   = true
    contexts = var.branch_protection_conf.contexts
  }

  required_pull_request_reviews {
    require_code_owner_reviews      = false
    required_approving_review_count = var.branch_protection_conf.review_count

  }
}