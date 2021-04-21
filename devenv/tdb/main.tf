terraform {
  required_version = ">= 0.12"
  backend "s3" {
    bucket = "terraform-state-devenv"
    key    = "denenv"
    region = "eu-central-1"
  }
}

provider "aws" {
  # 3.0.0 seems to have bug in fetching the ecs iam role
  #version = "= 2.70"
  region = data.terraform_remote_state.base.outputs.region
}

locals {
  common_tags = map(
    "managed", "automation",
    "ou", "devops",
    "purpose", "test",
    "env", var.env_name
  )
}

# For VPC
data "terraform_remote_state" "infra" {
  backend = "remote"

  config = {
    organization = "Tyk"
    workspaces = {
      name = var.infra
    }
  }
}

# For region
data "terraform_remote_state" "base" {
  backend = "remote"

  config = {
    organization = "Tyk"
    workspaces = {
      name = var.base
    }
  }
}

data "aws_iam_role" "ecs_task_execution_role" {
  name = "ecsExecutionRole"
}

# Private subnets
data "aws_subnet_ids" "private" {
  vpc_id = data.terraform_remote_state.infra.outputs.vpc_id

  tags = {
    Type = "private"
  }
}

# Public subnets
data "aws_subnet_ids" "public" {
  vpc_id = data.terraform_remote_state.infra.outputs.vpc_id

  tags = {
    Type = "public"
  }
}

