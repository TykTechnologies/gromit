# Prerequisites

As per each repository, please first import the existent resources
```
terraform import module.tyk.github_repository.repository tyk
terraform import module.tyk.github_branch.default tyk:master
terraform import module.tyk.github_branch_default.default tyk
```

Export github credentials using github PAT

```
export GITHUB_TOKEN="asdjkjsdcjxckzxkcxxxcka"
```

# How to Run

```
terraform init
terraform plan
terraform apply
```

Terraform plan should look like this

```
Acquiring state lock. This may take a few moments...
module.tyk.github_repository.repository: Refreshing state... [id=tyk]
module.tyk.github_branch.default: Refreshing state... [id=tyk:master]
module.tyk.github_branch_default.default: Refreshing state... [id=tyk]

Terraform used the selected providers to generate the following execution plan. Resource actions are indicated with the following symbols:
  + create
  ~ update in-place

Terraform will perform the following actions:

  # module.tyk.github_branch_protection.automerge will be created
  + resource "github_branch_protection" "automerge" {
      + allows_deletions                = false
      + allows_force_pushes             = false
      + blocks_creations                = false
      + enforce_admins                  = false
      + id                              = (known after apply)
      + pattern                         = "master"
      + repository_id                   = "tyk"
      + require_conversation_resolution = true
      + require_signed_commits          = true
      + required_linear_history         = false

      + required_pull_request_reviews {
          + require_code_owner_reviews      = true
          + required_approving_review_count = 2
        }

      + required_status_checks {
          + strict = true
        }
    }

  # module.tyk.github_repository.repository will be updated in-place
  ~ resource "github_repository" "repository" {
      ~ allow_auto_merge            = false -> true
        id                          = "tyk"
        name                        = "tyk"
        # (30 unchanged attributes hidden)
    }

Plan: 1 to add, 1 to change, 0 to destroy.

─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

Note: You didn't use the -out option to save this plan, so Terraform can't guarantee to take exactly these actions if you run "terraform apply" now.
Releasing state lock. This may take a few moments...
```