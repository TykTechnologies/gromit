terraform {

  # Being used until TFCloud can be used
  backend "s3" {
    bucket         = "terraform-state-devenv"
    key            = "github-policy"
    region         = "eu-central-1"
    dynamodb_table = "terraform-state-locks"
  }

  required_providers {
    github = {
      source  = "integrations/github"
      version = ">= 5.5.0"
    }
  }
}

provider "github" {
  owner = "TykTechnologies"
}

module "portal" {
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
  release_branches = [
    { branch    = "master",
      reviewers = 1,
      convos    = false,
    required_tests = ["test (1.16.x, ubuntu-latest, amd64, 15.x)"] },
  ]
}

module "tyk-analytics-ui" {
  source                      = "../../../modules/github-repos"
  repo                        = "tyk-analytics-ui"
  description                 = "User interface for our dashboard. Backend: https://github.com/TykTechnologies/tyk-analytics"
  topics                      = []
  visibility                  = "private"
  default_branch              = "master"
  vulnerability_alerts        = true
  squash_merge_commit_message = "PR_BODY"
  squash_merge_commit_title   = "PR_TITLE"
  release_branches = var.tyk_release_branches
}

module "tyk-analytics" {
  source                      = "../../../modules/github-repos"
  repo                        = "tyk-analytics"
  description                 = "Tyk Dashboard New Repository"
  topics                      = []
  visibility                  = "private"
  default_branch              = "master"
  vulnerability_alerts        = true
  squash_merge_commit_message = "PR_BODY"
  squash_merge_commit_title   = "PR_TITLE"
  release_branches = [
    { branch    = "master",
      reviewers = 2,
      convos    = false,
    required_tests = ["commit message linter", "test (1.16.x, ubuntu-latest, amd64, 15.x)", "sqlite", "ci", "mongo"] },
    { branch        = "release-4.3",
      reviewers     = 0,
      convos        = false,
      source_branch = "release-4",
    required_tests = ["commit message linter", "test (1.16.x, ubuntu-latest, amd64, 15.x)", "sqlite", "ci", "mongo"] },
    { branch        = "release-4.3.0",
      reviewers     = 0,
      convos        = false,
      source_branch = "release-4",
    required_tests = ["commit message linter", "test (1.16.x, ubuntu-latest, amd64, 15.x)", "sqlite", "ci", "mongo"] },
  ]
}

module "tyk-identity-broker" {
  source               = "../../../modules/github-repos"
  repo                 = "tyk-identity-broker"
  description          = "Tyk Authentication Proxy for third-party login"
  topics               = []
  default_branch       = "master"
  vulnerability_alerts = true
  release_branches = [
    { branch    = "master",
      reviewers = 1,
      convos    = false,
    required_tests = [] },
  ]
}

module "tyk-pump" {
  source               = "../../../modules/github-repos"
  repo                 = "tyk-pump"
  description          = "Tyk Analytics Pump to move analytics data from Redis to any supported back end (multiple back ends can be written to at once)."
  topics               = []
  wiki                 = false
  default_branch       = "master"
  vulnerability_alerts = true
  release_branches = [
    { branch    = "master",
      reviewers = 2,
      convos    = false,
    required_tests = [] },
  ]
}

module "tyk-sink" {
  source               = "../../../modules/github-repos"
  repo                 = "tyk-sink"
  description          = "Tyk RPC Server backend (bridge)"
  topics               = []
  visibility           = "private"
  default_branch       = "master"
  merge_commit         = true
  rebase_merge         = true
  vulnerability_alerts = false
  release_branches = [
    { branch    = "master",
      reviewers = 1,
      convos    = false,
    required_tests = [] },
  ]
}

module "tyk" {
  source               = "../../../modules/github-repos"
  repo                 = "tyk"
  description          = "Tyk Open Source API Gateway written in Go, supporting REST, GraphQL, TCP and gRPC protocols"
  topics               = ["api", "api-gateway", "api-management", "cloudnative", "go", "graphql", "grpc", "k8s", "kubernetes", "microservices", "reverse-proxy", "tyk"]
  wiki                 = false
  default_branch       = "master"
  vulnerability_alerts = true
  release_branches = [
    { branch         = "master",
      reviewers      = 2,
      required_tests = ["Go 1.16 Redis 5"],
    convos = false },
    { branch         = "release-4.3",
      reviewers      = 0,
      source_branch  = "release-4",
      required_tests = ["Go 1.16 Redis 5"],
    convos = false },
    { branch         = "release-4.3.0",
      reviewers      = 0,
      source_branch  = "release-4",
      required_tests = ["Go 1.16 Redis 5"],
    convos = false },
  ]
}