module "tyk-analytics" {
  source                      = "../../../modules/github-repos"
  repo                   = "{{ .Name }}"
  description            = "{{ .Description }}"
  topics                      = []
  visibility                  = "private"
  default_branch              = "master"
  vulnerability_alerts        = true
  squash_merge_commit_message = "PR_BODY"
  squash_merge_commit_title   = "PR_TITLE"
  release_branches = [
    { branch    = "master",
      reviewers = 2,
      convos    = false,
    required_tests = ["commit message linter", "test (1.16.x, ubuntu-latest, amd64, 15.x)", "sqlite", "ci", "mongo"] },
    { branch        = "release-4.3",
      reviewers     = 0,
      convos        = false,
      source_branch = "release-4",
    required_tests = ["commit message linter", "test (1.16.x, ubuntu-latest, amd64, 15.x)", "sqlite", "ci", "mongo"] },
    { branch        = "release-4.3.0",
      reviewers     = 0,
      convos        = false,
      source_branch = "release-4",
    required_tests = ["commit message linter", "test (1.16.x, ubuntu-latest, amd64, 15.x)", "sqlite", "ci", "mongo"] },
  ]
}