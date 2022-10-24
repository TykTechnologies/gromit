variable "repo" {
  type        = string
  description = "Repository name"
}

variable "description" {
  type        = string
  description = "Repository description"
}

variable "visibility" {
  type        = string
  description = "Repository visibility , private or public"
  default     = "public"
}

variable "wiki" {
  type        = bool
  description = "Repository has wiki enabled or not"
  default     = true
}

variable "topics" {
  type        = list(string)
  description = "Github topics"
}

variable "default_branch" {
  type        = string
  description = "Repository default branch name"
}

variable "branch_protection_conf_set" {
  type = set(object({
    pattern             = string
    signed_commits      = bool
    linear_history      = bool
    allows_deletions    = bool
    allows_force_pushes = bool
    blocks_creations    = bool
    contexts            = list(string)
    review_count        = number
  }))
}