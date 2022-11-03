terraform {

  required_providers {
    github = {
      source  = "integrations/github"
      version = "5.5.0"
    }
  }

}

resource "github_repository" "repository" {
  name                        = var.repo
  description                 = var.description
  visibility                  = var.visibility
  allow_rebase_merge          = var.rebase_merge
  allow_squash_merge          = true
  squash_merge_commit_message = var.squash_merge_commit_message
  squash_merge_commit_title   = var.squash_merge_commit_title
  allow_merge_commit          = var.merge_commit
  allow_auto_merge            = true
  delete_branch_on_merge      = var.delete_branch_on_merge
  vulnerability_alerts        = var.vulnerability_alerts
  has_downloads               = true
  has_issues                  = true
  has_wiki                    = var.wiki
  has_projects                = true
  topics                      = var.topics
}

resource "github_branch" "default" {
  repository = github_repository.repository.name
  branch     = var.default_branch
}

resource "github_branch" "release_branches" {
  for_each = toset(var.release_branches)
  repository = github_repository.repository.name
  branch     = each.value
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
    push_restrictions   = each.value.push_restrictions
    contexts            = each.value.contexts
    review_count        = each.value.review_count
  }

}