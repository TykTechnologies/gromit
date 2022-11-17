module "tyk-pump" {
  source               = "../../../modules/github-repos"
  repo                   = "{{ .Name }}"
  description            = "{{ .Description }}"
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