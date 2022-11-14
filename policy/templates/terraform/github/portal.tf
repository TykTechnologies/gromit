module "portal" {
  source                 = "../../../modules/github-repos"
  repo                   = "portal"
  description            = "Portal is a full-featured developer portal, blog and CMS"
  topics                 = ["portal", "api-gateway"]
  visibility             = "private"
  default_branch         = "master"
  merge_commit           = true
  rebase_merge           = true
  delete_branch_on_merge = false
  vulnerability_alerts   = false
  release_branches = [
    { branch    = "master",
      reviewers = 1,
      convos    = false,
    required_tests = ["test (1.16.x, ubuntu-latest, amd64, 15.x)"] },
  ]
}