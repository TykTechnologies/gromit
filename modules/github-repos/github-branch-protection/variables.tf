variable "repo" {
  type        = string
  description = "Repository name"
}

variable "branch_protection_conf" {
  type = object({
    pattern             = string
    signed_commits      = bool
    linear_history      = bool
    allows_deletions    = bool
    allows_force_pushes = bool
    blocks_creations    = bool
    push_restrictions   = list(string)
    contexts            = list(string)
    review_count        = number
  })
}