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

# This is exported in outputs.tf
data "aws_region" "current" {}

resource "aws_ecr_repository" "integration" {
  for_each = toset(local.tyk_repos)
  
  name                 = each.key
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = false
  }

  tags = local.common_tags
}

resource "aws_ecr_lifecycle_policy" "retain_2w" {
  for_each = toset(local.tyk_repos)

  repository = each.key

  policy = <<EOF
{
    "rules": [
        {
            "rulePriority": 1,
            "description": "Expire untagged images older than 1 week",
            "selection": {
                "tagStatus": "untagged",
                "countType": "sinceImagePushed",
                "countUnit": "days",
                "countNumber": 7
            },
            "action": {
                "type": "expire"
            }
        },
        {
            "rulePriority": 2,
            "description": "Expire all images older than 2 weeks",
            "selection": {
                "tagStatus": "any",
                "countType": "sinceImagePushed",
                "countUnit": "days",
                "countNumber": 14
            },
            "action": {
                "type": "expire"
            }
        }
    ]
}
EOF
}

# terraform apply -target=null_resource.debug will show the rendered template
# resource "null_resource" "debug" {
#   triggers = {
#     json = "${data.template_file.tyk_repo_access.rendered}"
#   }
# }

