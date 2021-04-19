terraform {
  required_version = ">= 0.13"
  backend "s3" {
    bucket         = "terraform-state-devenv"
    key            = "test"
    region         = "eu-central-1"
    # dynamodb_table = "terraform-state-locks"
  }
}

provider "aws" {
  region = "eu-central-1"
}

# Internal variables
locals {
  # name should match the tf workspace name
  name = "test"
  # Repositories to create
  tyk_repos = ["tyk", "tyk-analytics", "tyk-pump" ]
  gromit = {
    "domain" = "test.tyk.technology"
  }
  common_tags = {
    "managed" = "automation",
    "ou"      = "devops",
    "purpose" = "ci-test",
    "env"     = local.name
  }
}

resource "aws_ecr_repository" "integration_test" {
  for_each = toset(local.tyk_repos)
  
  name                 = each.key
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = false
  }

  tags = local.common_tags
}

resource "aws_route53_zone" "test_tyk_tech" {
  name = local.gromit.domain
  comment = "Hosted zone for gromit testing"

  tags = local.common_tags
}
