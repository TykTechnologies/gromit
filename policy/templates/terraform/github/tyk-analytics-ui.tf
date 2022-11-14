module "tyk-analytics-ui" {
  source                      = "../../../modules/github-repos"
  repo                        = "tyk-analytics-ui"
  description                 = "User interface for our dashboard. Backend: https://github.com/TykTechnologies/tyk-analytics"
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
    required_tests = ["test (1.16.x, ubuntu-latest, amd64, 15.x)", "test"] },
    { branch        = "release-4.3",
      reviewers     = 0,
      convos        = false,
      source_branch = "release-4",
    required_tests = ["test (1.16.x, ubuntu-latest, amd64, 15.x)", "test"] },
    { branch        = "release-4.3.0",
      reviewers     = 0,
      convos        = false,
      source_branch = "release-4",
    required_tests = ["test (1.16.x, ubuntu-latest, amd64, 15.x)", "test"] },
  ]
}