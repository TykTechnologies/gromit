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
  visibility             = var.visibility
  allow_rebase_merge     = false
  allow_squash_merge     = true
  allow_merge_commit     = false
  allow_auto_merge       = true
  delete_branch_on_merge = true
  has_downloads          = true
  has_issues             = true
  has_wiki               = var.wiki
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

module "protected_branches" {
  for_each = { for branch in var.branch_protection_conf_set : branch.pattern => branch }
  source   = "./github-branch-protection"
  repo     = github_repository.repository.node_id
  branch_protection_conf = {
    pattern             = each.value.pattern
    signed_commits      = each.value.signed_commits
    linear_history      = each.value.linear_history
    allows_deletions    = each.value.allows_deletions
    allows_force_pushes = each.value.allows_force_pushes
    blocks_creations    = each.value.blocks_creations
    contexts            = each.value.contexts
    review_count        = each.value.review_count
  }

}