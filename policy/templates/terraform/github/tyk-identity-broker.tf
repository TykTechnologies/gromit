module "tyk-identity-broker" {
  source               = "../../../modules/github-repos"
  repo                   = "{{ .Name }}"
  description            = "{{ .Description }}"
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