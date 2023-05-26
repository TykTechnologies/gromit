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
  topics                      = [{{ range $index, $topic := .Topics }}{{ if $index }},{{ end }}"{{ $topic }}"{{ end }}]
  visibility                  = "{{.Visibility}}"
  wiki                        = {{ .Wiki }}
  vulnerability_alerts        = {{ .VulnerabilityAlerts }}
  squash_merge_commit_message = "{{ .SquashMsg }}"
  squash_merge_commit_title   = "{{ .SquashTitle }}"
  {{- if or (eq .Name "tyk-sink") (eq .Name "portal") }}
  merge_commit = true
  rebase_merge = true
  {{- end }}
  {{- if eq .Name "portal" }}
  delete_branch_on_merge = false
  {{- end }}
  release_branches     = concat(var.historical_branches,[
{{- range $branch, $values := .ActiveReleaseBranches }}
{ branch    = "{{ $branch }}",
	reviewers = "{{ $values.ReviewCount }}",
	convos    = "{{ $values.Convos }}",
	{{- if $values.SourceBranch }}
	source_branch  = "{{ $values.SourceBranch }}",
	{{- end }}
	required_tests = [{{ range $index, $test := $values.Tests }}{{ if $index }},{{ end }}"{{ $test }}"{{ end }}]},
{{- end }}
])
}
