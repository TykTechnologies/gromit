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
  common_tags = {
    "managed" = "automation",
    "ou"      = "devops",
    "purpose" = "ci",
    "env"     = local.name
  }
}

resource "aws_ecr_repository" "integration" {
  for_each = toset(local.tyk_repos)
  
  name                 = each.key
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = false
  }

  tags = local.common_tags
}
