terraform {
  required_providers {
    github = {
      source  = "integrations/github"
      version = "5.5.0"
    }
  }

  required_version = ">= 1.0.10"

  backend "s3" {
    bucket = "terraform-state-devenv"
    key    = "devenv"
    region = "eu-central-1"
    dynamodb_table = "terraform-state-locks"
  }

}


provider "github" {
  # set gh_token if GITHUB_TOKEN is not present locally.
  #token = var.gh_token
  owner = "tyklabs"
  #base_url = "https://github.com/tyklabs/"
}



resource "github_repository" "tyk" {
  name               = "tyk"
  description        = "Fork of the Tyk GW for experiments"
  visibility         = "public"
  allow_rebase_merge = false
  allow_squash_merge = true
  allow_merge_commit = false
  allow_auto_merge   = true

}

resource "github_branch" "master" {
  repository = github_repository.tyk.name
  branch     = "master"
}

resource "github_branch_default" "default" {
  repository = github_repository.tyk.name
  branch     = github_branch.master.branch
}

resource "github_branch_protection" "automerge" {
  repository_id = github_repository.tyk.name
  pattern       = "master"

  #checks for automerge
  require_signed_commits          = true
  require_conversation_resolution = true

  required_status_checks {
    strict   = true
    contexts = []
  }

  required_pull_request_reviews {
    require_code_owner_reviews      = true
    required_approving_review_count = 2
  }
}