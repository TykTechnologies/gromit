module "tyk-sink" {
  source               = "../../../modules/github-repos"
  repo                 = "tyk-sink"
  description          = "Tyk RPC Server backend (bridge)"
  topics               = []
  visibility           = "private"
  default_branch       = "master"
  merge_commit         = true
  rebase_merge         = true
  vulnerability_alerts = false
  release_branches = [
    { branch    = "master",
      reviewers = 1,
      convos    = false,
    required_tests = [] },
  ]
}