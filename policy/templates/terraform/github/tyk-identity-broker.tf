module "tyk-identity-broker" {
  source               = "../../../modules/github-repos"
  repo                 = "tyk-identity-broker"
  description          = "Tyk Authentication Proxy for third-party login"
  topics               = []
  default_branch       = "master"
  vulnerability_alerts = true
  release_branches = [
    { branch    = "master",
      reviewers = 1,
      convos    = false,
    required_tests = [] },
  ]
}