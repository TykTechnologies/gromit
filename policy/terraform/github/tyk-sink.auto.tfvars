tyk-sink_release_branches = [
  { branch    = "master",
    reviewers = 1,
    convos    = false,
  required_tests = [
    "1.16",
    "ci"]
  },
  { branch    = "release-2",
    reviewers = 1,
    convos    = false,
  required_tests = [
    "1.16",
    "ci"]
  },
]