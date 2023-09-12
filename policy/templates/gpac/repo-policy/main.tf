terraform {

  #Being used until TFCloud can be used
  backend "remote" {
    hostname     = "app.terraform.io"
    organization = "Tyk"
    workspaces {
      name = "repo-policy-{{ .Name }}"
    }
  }

  required_providers {
    github = {
      source  = "integrations/github"
    }
  }
}

provider "github" {
  owner = "TykTechnologies"
}

variable "topics" {
  type = list(string)
  description = "Tag cloud under About"
}

variable "visibility" {
  type = string
  description = "public or private"
  validation {
    condition = var.visibility == "public" || var.visibility == "private"
    error_message = "Repository can be either public or private"
  }
}

variable "wiki" {
  type = bool
  description = "Enable wiki"
}

variable "vulnalerts" {
  type = bool
  description = "Security and analysis features"
}

variable "squashtitle" {
  type = string
  default = "PR_TITLE"
  description = "Commit title of squashed PR"
}

variable "squashmsg" {
  type = string
  default = "PR_BODY"
  description = "Commit body of squashed PR"
}

# Copypasta from modules/github-repos/variables.tf
# FIXME: Unmodularise the github-repos module
variable "historical_branches" {
  type = list(object({
    branch         = string           # Name of the branch
    source_branch  = optional(string) # Source of the branch, needed when creating it
    reviewers      = number           # Min number of reviews needed
    required_tests = list(string)     # Workflows that need to pass before merging
    convos         = bool             # Should conversations be resolved before merging

  }))
  description = "List of branches managed by terraform"
}

module "{{ .Name }}" {
  source               = "./modules/github-repos"
  repo                 = "{{ .Name }}"
  description          = "{{ .Description }}"
  default_branch       = "{{ .Default }}"
  topics                      = var.topics
  visibility                  = var.visibility
  wiki                        = var.wiki
  vulnerability_alerts        = var.vulnlerts
  squash_merge_commit_message = var.squashmsg
  squash_merge_commit_title   = var.squashtitle
  {{- if or (eq .Name "tyk-sink") (eq .Name "portal") }}
  merge_commit = true
  rebase_merge = true
  {{- end }}
  {{- if eq .Name "portal" }}
  delete_branch_on_merge = false
  {{- end }}
  release_branches     = concat(var.historical_branches,[
{{- range $branch, $values := .Branches }}
{ branch    = "{{ $branch }}",
	reviewers = "{{ $values.ReviewCount }}",
	convos    = "{{ $values.Convos }}",
	{{- if $values.SourceBranch }}
	source_branch  = "{{ $values.SourceBranch }}",
	{{- end }}
	required_tests = [{{ range $index, $test := $values.Tests }}{{ if $index }},{{ end }}"{{ $test }}"{{ end }}]},
    {{- end }}{{/* range */}}
])
}
