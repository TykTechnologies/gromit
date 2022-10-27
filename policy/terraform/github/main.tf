locals {
  repos = ["tyk", "tyk-analytics", "tyk-pump", "tyk-sink", "tyk-identity-broker", "portal"]
}

terraform {

  backend "s3" {
    bucket         = "terraform-state-devenv"
    key            = "github-policy"
    region         = "eu-central-1"
    dynamodb_table = "terraform-state-locks"
  }

  # backend "remote" {
  #   hostname     = "app.terraform.io"
  #   organization = "Tyk"
  #   workspaces {
  #     name = "github-policy"
  #   }
  # }

  required_providers {
    github = {
      source  = "integrations/github"
      version = "5.5.0"
    }
  }

  required_version = ">= 1.0.10"
}

provider "github" {
  # set gh_token if GITHUB_TOKEN is not present locally.
  #token = var.gh_token
  owner = "TykTechnologies"
  # organization = "TykTechnologies"
  #base_url = "https://github.com/TykTechnologies"
}

module "tyk" {
  # source                          = "git::https://github.com/TykTechnologies/gromit.git//modules/github-repos?ref=feat/td-1220/github-PaC-terraform"
  source               = "../../../modules/github-repos"
  repo                 = "tyk"
  description          = "Tyk Open Source API Gateway written in Go, supporting REST, GraphQL, TCP and gRPC protocols"
  topics               = ["api", "api-gateway", "api-management", "cloudnative", "go", "graphql", "grpc", "k8s", "kubernetes", "microservices", "reverse-proxy", "tyk"]
  wiki                 = false
  default_branch       = "master"
  vulnerability_alerts = true
  branch_protection_conf_set = [{
    pattern             = "master"
    signed_commits      = false
    linear_history      = false
    allows_deletions    = false
    allows_force_pushes = false
    blocks_creations    = false
    push_restrictions   = []
    contexts = [
      # "test",
      "Go 1.16 Redis 5"
      #   "Analyze (go)",
      #   "1.16",
      #   "lint",
      #   "1.16-el7",
      #   "ci",
      #   "upgrade-deb (amd64, ubuntu:xenial)",
      #   " upgrade-deb (amd64, ubuntu:bionic)",
      #   "upgrade-deb (amd64, ubuntu:focal)",
      #   "upgrade-deb (amd64, debian:bullseye)",
      #   "upgrade-deb (arm64, ubuntu:xenial)",
      #   "upgrade-deb (arm64, ubuntu:bionic)",
      #   " upgrade-deb (arm64, ubuntu:focal)",
      #   "upgrade-deb (arm64, debian:bullseye)",
      #   "upgrade-rpm (ubi7/ubi)",
      #   "upgrade-rpm (ubi8/ubi)",
      #   "smoke-tests",
      #   "CodeQL",
      #   "SonarCloud",
      # "SonarCloud Code Analysis"
    ]
    review_count = 2
    },
    {
      pattern             = "release-3.2"
      signed_commits      = false
      linear_history      = true
      allows_deletions    = false
      allows_force_pushes = false
      blocks_creations    = false
      push_restrictions   = []
      contexts            = []
      review_count        = 2
    },
    {
      pattern             = "release-3-lts"
      signed_commits      = false
      linear_history      = true
      allows_deletions    = true
      allows_force_pushes = false
      blocks_creations    = false
      push_restrictions   = []
      contexts            = []
      review_count        = 2
    }
    # {
    #   pattern             = "release-2.9"
    #   signed_commits      = false
    #   linear_history      = true
    #   allows_deletions    = false
    #   allows_force_pushes = false
    #   blocks_creations    = false
    #   push_restrictions   = []
    #   contexts            = []
    #   review_count        = 2
    # }
  ]
}

module "tyk-analytics" {
  # source                          = "git::https://github.com/TykTechnologies/gromit.git//modules/github-repos?ref=feat/td-1220/github-PaC-terraform"
  source                      = "../../../modules/github-repos"
  repo                        = "tyk-analytics"
  description                 = "Tyk Dashboard New Repository"
  topics                      = []
  visibility                  = "private"
  default_branch              = "master"
  vulnerability_alerts        = true
  squash_merge_commit_message = "PR_BODY"
  squash_merge_commit_title   = "PR_TITLE"
  branch_protection_conf_set = [{
    pattern             = "master"
    signed_commits      = false
    linear_history      = false
    allows_deletions    = false
    allows_force_pushes = false
    blocks_creations    = false
    push_restrictions   = []
    contexts = [
      "commit message linter",
      "test (1.16.x, ubuntu-latest, amd64, 15.x)",
      "sqlite",
      "ci",
      "mongo"
    ],
    review_count = 2
    },
    {
      pattern             = "stable"
      signed_commits      = false
      linear_history      = false
      allows_deletions    = false
      allows_force_pushes = false
      blocks_creations    = false
      push_restrictions   = ["MDQ6VXNlcjE0MDA5", "MDQ6VXNlcjI0NDQ0MDk="]
      contexts            = []
      review_count        = 2
    },
    {
      pattern             = "target/cloud"
      signed_commits      = false
      linear_history      = false
      allows_deletions    = false
      allows_force_pushes = false
      blocks_creations    = false
      push_restrictions   = ["MDQ6VXNlcjE0MDA5", "MDQ6VXNlcjI0NDQ0MDk="]
      contexts            = []
      review_count        = 2
    },
    {
      pattern             = "target/stage"
      signed_commits      = false
      linear_history      = false
      allows_deletions    = false
      allows_force_pushes = false
      blocks_creations    = false
      push_restrictions   = ["MDQ6VXNlcjE0MDA5", "MDQ6VXNlcjI0NDQ0MDk="]
      contexts            = []
      review_count        = 2
    }
  ]
}

module "tyk-pump" {
  # source                          = "git::https://github.com/TykTechnologies/gromit.git//modules/github-repos?ref=feat/td-1220/github-PaC-terraform"
  source               = "../../../modules/github-repos"
  repo                 = "tyk-pump"
  description          = "Tyk Analytics Pump to move analytics data from Redis to any supported back end (multiple back ends can be written to at once)."
  topics               = []
  wiki                 = false
  default_branch       = "master"
  vulnerability_alerts = true
  branch_protection_conf_set = [{
    pattern             = "master"
    signed_commits      = false
    linear_history      = false
    allows_deletions    = false
    allows_force_pushes = false
    blocks_creations    = false
    push_restrictions   = []
    contexts            = []
    review_count        = 2
    }
    ,
    {
      pattern             = "stable"
      signed_commits      = false
      linear_history      = false
      allows_deletions    = false
      allows_force_pushes = false
      blocks_creations    = false
      push_restrictions   = []
      contexts            = []
      review_count        = 2
    },
    {
      pattern             = "target/cloud"
      signed_commits      = false
      linear_history      = false
      allows_deletions    = false
      allows_force_pushes = false
      blocks_creations    = false
      push_restrictions   = ["MDQ6VXNlcjE0MDA5", "MDQ6VXNlcjI0NDQ0MDk="]
      contexts            = []
      review_count        = 2
    },
    {
      pattern             = "target/stage"
      signed_commits      = false
      linear_history      = false
      allows_deletions    = false
      allows_force_pushes = false
      blocks_creations    = false
      push_restrictions   = ["MDQ6VXNlcjE0MDA5", "MDQ6VXNlcjI0NDQ0MDk="]
      contexts            = []
      review_count        = 2
    }
  ]
}

module "tyk-sink" {
  # source                          = "git::https://github.com/TykTechnologies/gromit.git//modules/github-repos?ref=feat/td-1220/github-PaC-terraform"
  source               = "../../../modules/github-repos"
  repo                 = "tyk-sink"
  description          = "Tyk RPC Server backend (bridge)"
  topics               = []
  visibility           = "private"
  default_branch       = "master"
  merge_commit         = true
  rebase_merge         = true
  vulnerability_alerts = false

  branch_protection_conf_set = [
    # {
    #   pattern             = "master"
    #   signed_commits      = false
    #   linear_history      = false
    #   allows_deletions    = false
    #   allows_force_pushes = false
    #   blocks_creations    = false
    #   push_restrictions   = []
    #   contexts            = []
    #   review_count        = 2
    # }
  ]
}

module "tyk-identity-broker" {
  # source                          = "git::https://github.com/TykTechnologies/gromit.git//modules/github-repos?ref=feat/td-1220/github-PaC-terraform"
  source               = "../../../modules/github-repos"
  repo                 = "tyk-identity-broker"
  description          = "Tyk Authentication Proxy for third-party login"
  topics               = []
  default_branch       = "master"
  vulnerability_alerts = true
  branch_protection_conf_set = [{
    pattern             = "master"
    signed_commits      = false
    linear_history      = false
    allows_deletions    = false
    allows_force_pushes = false
    blocks_creations    = false
    push_restrictions   = []
    contexts            = []
    review_count        = 1
  }]
}

module "portal" {
  # source                          = "git::https://github.com/TykTechnologies/gromit.git//modules/github-repos?ref=feat/td-1220/github-PaC-terraform"
  source                 = "../../../modules/github-repos"
  repo                   = "portal"
  description            = "Portal is a full-featured developer portal, blog and CMS"
  topics                 = ["portal", "api-gateway"]
  visibility             = "private"
  default_branch         = "master"
  merge_commit           = true
  rebase_merge           = true
  delete_branch_on_merge = false
  vulnerability_alerts   = false

  branch_protection_conf_set = [
  #  {
  #   pattern             = "master"
  #   signed_commits      = false
  #   linear_history      = false
  #   allows_deletions    = false
  #   allows_force_pushes = false
  #   blocks_creations    = false
  #   push_restrictions   = []
  #   contexts            = []
  #   review_count        = 2
  # }
  ]
}