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

variable "merge_commit" {
  type        = bool
  description = "Set to false to disable merge commits on the repository"
  default     = false
}

variable "rebase_merge" {
  type        = bool
  description = "Set to false to disable rebase merges on the repository"
  default     = false
}

variable "delete_branch_on_merge" {
  type        = bool
  description = "Automatically delete head branch after a pull request is merged"
  default     = true
}

variable "vulnerability_alerts" {
  type        = bool
  description = "Set to true to enable security alerts for vulnerable dependencies. Enabling requires alerts to be enabled on the owner level. (Note for importing: GitHub enables the alerts on public repos but disables them on private repos by default.)"
  default     = true
}

variable "squash_merge_commit_message" {
  type        = string
  description = "Can be PR_BODY, COMMIT_MESSAGES, or BLANK for a default squash merge commit message."
  default     = "COMMIT_MESSAGES"
}

variable "squash_merge_commit_title" {
  type        = string
  description = "Can be PR_TITLE or COMMIT_OR_PR_TITLE for a default squash merge commit title."
  default     = "COMMIT_OR_PR_TITLE"
}


variable "branch_protection_conf_set" {
  type = set(object({
    pattern             = string
    signed_commits      = bool
    linear_history      = bool
    allows_deletions    = bool
    allows_force_pushes = bool
    blocks_creations    = bool
    push_restrictions   = list(string)
    contexts            = list(string)
    review_count        = number
  }))
}