variable "repo" {
  type        = string
  description = "Repository name"
}

variable "description" {
  type        = string
  description = "Repository description"
}

variable "topics" {
  type        = list(string)
  description = "Github topics"
}

variable "default_branch" {
  type        = string
  description = "Repository default branch name"
}

variable "required_status_checks_contexts" {
  type        = list(string)
  description = "Required status checks"
}

variable "required_approving_review_count" {
  type        = number
  description = "Number of required PR reviewers for approval"
  default     = 2
}