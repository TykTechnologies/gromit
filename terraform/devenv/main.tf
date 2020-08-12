terraform {
  required_version = ">= 0.12"
  backend "s3" {
    bucket         = "terraform-state-devenv"
    key            = "devenv"
    region         = "eu-central-1"
    dynamodb_table = "terraform-state-locks"
  }
}

provider "aws" {
  # 3.0.0 seems to have bug in fetching the ecs iam role
  version = "= 2.70"
  region  = data.terraform_remote_state.base.outputs.region
}

# Internal variables

locals {
  common_tags = "${map(
    "managed", "automation",
    "ou", "devops",
    "purpose", "ci",
    "env", var.name,
  )}"
  # Name for the task
  gw_name    = join("-", [var.name, "gw"])
  db_name    = join("-", [var.name, "db"])
  pump_name  = join("-", [var.name, "pump"])
  redis_name = join("-", [var.name, "redis"])
  int_domain = join(".", [var.name, "internal"])
  # Construct full ECR URLs
  tyk_image           = join(":", [data.terraform_remote_state.base.outputs.tyk["ecr"], var.tyk])
  tyk-analytics_image = join(":", [data.terraform_remote_state.base.outputs.tyk-analytics["ecr"], var.tyk-pump])
  tyk-pump_image      = join(":", [data.terraform_remote_state.base.outputs.tyk-pump["ecr"], var.tyk-pump])
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

# EFS, ECR

data "terraform_remote_state" "base" {
  backend = "remote"

  config = {
    organization = "Tyk"
    workspaces = {
      name = var.base
    }
  }
}

# ECS cluster

resource "aws_ecs_cluster" "env" {
  name = var.name

  setting {
    name  = "containerInsights"
    value = "enabled"
  }
  tags = local.common_tags
}

data "aws_iam_role" "ecs_task_execution_role" {
  name = "ecsExecutionRole"
}

# Security groups

resource "aws_security_group" "gateway" {
  name        = "${var.name}-gateway"
  description = "Traffic from anywhere 8000-9000"
  vpc_id      = data.terraform_remote_state.infra.outputs.vpc_id


  ingress {
    from_port   = 8000
    to_port     = 9000
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = local.common_tags
}

resource "aws_security_group" "dashboard" {
  name        = "${var.name}-dashboard"
  description = "Traffic from anywhere on 3000"
  vpc_id      = data.terraform_remote_state.infra.outputs.vpc_id


  ingress {
    from_port   = 3000
    to_port     = 3000
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = local.common_tags
}

resource "aws_security_group" "pump" {
  name        = "${var.name}-pump"
  description = "Allow traffic from anywhere in the vpc"
  vpc_id      = data.terraform_remote_state.infra.outputs.vpc_id


  ingress {
    from_port   = 3000
    to_port     = 3000
    protocol    = "tcp"
    cidr_blocks = [ data.terraform_remote_state.infra.outputs.vpc_cidr ]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = local.common_tags
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

# Private DNS
# Service discovery
resource "aws_service_discovery_private_dns_namespace" "internal" {
  name        = local.int_domain
  vpc         = data.terraform_remote_state.infra.outputs.vpc_id
  description = "The tyk conf files can use friendly names"
}
