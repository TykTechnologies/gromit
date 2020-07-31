terraform {
  required_version = ">= 0.12"
  backend "remote" {
    hostname     = "app.terraform.io"
    organization = "Tyk"

    workspaces {
      prefix = "env-"
    }
  }
}

provider "aws" {
  # 3.0.0 seems to have bug in fetching the ecs iam role
  version = "= 2.70"
  region  = var.region
}

# Internal variables

locals {
  common_tags = "${map(
    "managed", "automation",
    "ou", "devops",
    "purpose", "ci",
    "env", var.name_prefix,
  )}"
  # Name for the task
  gw_name    = join("-", [var.name_prefix, "gw"])
  db_name    = join("-", [var.name_prefix, "db"])
  pump_name  = join("-", [var.name_prefix, "pump"])
  redis_name = join("-", [var.name_prefix, "redis"])
  int_domain = join(".", [var.name_prefix, "internal"])
}

resource "aws_ecs_cluster" "env" {
  name = var.name_prefix

  tags = local.common_tags
}

# The default for ecs task definitions
data "aws_iam_role" "ecs_task_execution_role" {
  name = "ecsTaskExecutionRole"
}

# Security groups

resource "aws_security_group" "gateway" {
  name        = "gateway"
  description = "Traffic from anywhere 8000-9000"
  vpc_id      = data.aws_vpc.devenv.id


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
  name        = "dashboard"
  description = "Traffic from anywhere on 3000"
  vpc_id      = data.aws_vpc.devenv.id


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
  name        = "pump"
  description = "Allow traffic from anywhere in the vpc"
  vpc_id      = data.aws_vpc.devenv.id


  ingress {
    from_port   = 3000
    to_port     = 3000
    protocol    = "tcp"
    cidr_blocks = [data.aws_vpc.devenv.cidr_block]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = local.common_tags
}

# VPC that base and infra are running in

data "aws_vpc" "devenv" {
  id = var.vpc_id
}

# Private subnets

data "aws_subnet_ids" "private" {
  vpc_id = data.aws_vpc.devenv.id

  tags = {
    Type = "private"
  }
}

# Public subnets
data "aws_subnet_ids" "public" {
  vpc_id = data.aws_vpc.devenv.id

  tags = {
    Type = "public"
  }
}

# Private DNS
# Service discovery
resource "aws_service_discovery_private_dns_namespace" "internal" {
  name        = local.int_domain
  vpc         = data.aws_vpc.devenv.id
  description = "The tyk conf files can use friendly names"
}
