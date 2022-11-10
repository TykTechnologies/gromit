variable "tyk_release_branches" {
  type = list(object({
    branch         = string           # Name of the branch
    source_branch  = optional(string) # Source of the branch, needed when creating it
    reviewers      = number           # Min number of reviews needed
    required_tests = list(string)     # Workflows that need to pass before merging
    convos         = bool             # Should conversations be resolved before merging
  }))
  description = "List of tyk release branches managed by terraform"
}

variable "tyk-pump_release_branches" {
  type = list(object({
    branch         = string           # Name of the branch
    source_branch  = optional(string) # Source of the branch, needed when creating it
    reviewers      = number           # Min number of reviews needed
    required_tests = list(string)     # Workflows that need to pass before merging
    convos         = bool             # Should conversations be resolved before merging
  }))
  description = "List of tyk pump release branches managed by terraform"
}

variable "tyk-analytics_release_branches" {
  type = list(object({
    branch         = string           # Name of the branch
    source_branch  = optional(string) # Source of the branch, needed when creating it
    reviewers      = number           # Min number of reviews needed
    required_tests = list(string)     # Workflows that need to pass before merging
    convos         = bool             # Should conversations be resolved before merging
  }))
  description = "List of tyk analytics release branches managed by terraform"
}

variable "tyk-analytics-ui_release_branches" {
  type = list(object({
    branch         = string           # Name of the branch
    source_branch  = optional(string) # Source of the branch, needed when creating it
    reviewers      = number           # Min number of reviews needed
    required_tests = list(string)     # Workflows that need to pass before merging
    convos         = bool             # Should conversations be resolved before merging
  }))
  description = "List of tyk analytics ui release branches managed by terraform"
}

variable "tyk-identity-broker_release_branches" {
  type = list(object({
    branch         = string           # Name of the branch
    source_branch  = optional(string) # Source of the branch, needed when creating it
    reviewers      = number           # Min number of reviews needed
    required_tests = list(string)     # Workflows that need to pass before merging
    convos         = bool             # Should conversations be resolved before merging
  }))
  description = "List of tyk identity broker release branches managed by terraform"
}

variable "tyk-sink_release_branches" {
  type = list(object({
    branch         = string           # Name of the branch
    source_branch  = optional(string) # Source of the branch, needed when creating it
    reviewers      = number           # Min number of reviews needed
    required_tests = list(string)     # Workflows that need to pass before merging
    convos         = bool             # Should conversations be resolved before merging
  }))
  description = "List of tyk sink release branches managed by terraform"
}

variable "portal_release_branches" {
  type = list(object({
    branch         = string           # Name of the branch
    source_branch  = optional(string) # Source of the branch, needed when creating it
    reviewers      = number           # Min number of reviews needed
    required_tests = list(string)     # Workflows that need to pass before merging
    convos         = bool             # Should conversations be resolved before merging
  }))
  description = "List of new developer portal release branches managed by terraform"
}