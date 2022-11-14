module "tyk-pump" {
  source               = "../../../modules/github-repos"
  repo                 = "tyk-pump"
  description          = "Tyk Analytics Pump to move analytics data from Redis to any supported back end (multiple back ends can be written to at once)."
  topics               = []
  wiki                 = false
  default_branch       = "master"
  vulnerability_alerts = true
  release_branches = [
    { branch    = "master",
      reviewers = 2,
      convos    = false,
    required_tests = [] },
  ]
}