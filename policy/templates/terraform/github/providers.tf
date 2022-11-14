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