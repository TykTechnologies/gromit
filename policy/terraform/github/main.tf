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
  source                          = "../../../modules/github-repos"
  repo                            = "tyk"
  description                     = "Tyk Open Source API Gateway written in Go, supporting REST, GraphQL, TCP and gRPC protocols"
  topics                          = ["api", "api-gateway", "api-management", "cloudnative", "go", "graphql", "grpc", "k8s", "kubernetes", "microservices", "reverse-proxy", "tyk"]
  wiki                            = false
  default_branch                  = "master"
  required_status_checks_contexts = []
  required_approving_review_count = "2"
}

module "tyk-analytics" {
  # source                          = "git::https://github.com/TykTechnologies/gromit.git//modules/github-repos?ref=feat/td-1220/github-PaC-terraform"
  source                          = "../../../modules/github-repos"
  repo                            = "tyk-analytics"
  description                     = "Tyk Dashboard New Repository"
  topics                          = []
  visibility                      = "private"
  default_branch                  = "master"
  required_status_checks_contexts = []
  required_approving_review_count = "2"
}

module "tyk-pump" {
  # source                          = "git::https://github.com/TykTechnologies/gromit.git//modules/github-repos?ref=feat/td-1220/github-PaC-terraform"
  source                          = "../../../modules/github-repos"
  repo                            = "tyk-pump"
  description                     = "Tyk Analytics Pump to move analytics data from Redis to any supported back end (multiple back ends can be written to at once)."
  topics                          = []
  wiki                            = false
  default_branch                  = "master"
  required_status_checks_contexts = []
  required_approving_review_count = "2"
}

module "tyk-sink" {
  # source                          = "git::https://github.com/TykTechnologies/gromit.git//modules/github-repos?ref=feat/td-1220/github-PaC-terraform"
  source                          = "../../../modules/github-repos"
  repo                            = "tyk-sink"
  description                     = "Tyk Open Source API Gateway written in Go, supporting REST, GraphQL, TCP and gRPC protocols"
  topics                          = []
  visibility                      = "private"
  default_branch                  = "master"
  required_status_checks_contexts = []
  required_approving_review_count = "2"
}

module "tyk-identity-broker" {
  # source                          = "git::https://github.com/TykTechnologies/gromit.git//modules/github-repos?ref=feat/td-1220/github-PaC-terraform"
  source                          = "../../../modules/github-repos"
  repo                            = "tyk-identity-broker"
  description                     = "Tyk Authentication Proxy for third-party login"
  topics                          = []
  default_branch                  = "master"
  required_status_checks_contexts = []
  required_approving_review_count = "2"
}

module "portal" {
  # source                          = "git::https://github.com/TykTechnologies/gromit.git//modules/github-repos?ref=feat/td-1220/github-PaC-terraform"
  source                          = "../../../modules/github-repos"
  repo                            = "portal"
  description                     = "Portal is a full-featured developer portal, blog and CMS"
  topics                          = ["portal", "api-gateway"]
  visibility                      = "private"
  default_branch                  = "master"
  required_status_checks_contexts = []
  required_approving_review_count = "2"
}