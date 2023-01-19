portal_release_branches = [
  { branch    = "master",
    reviewers = 1,
    convos    = false,
  required_tests = [
    "test (1.16.x, ubuntu-latest, amd64, 15.x)",
    "1.15",
    "1.15-el7"]
  },
  { branch    = "release-1.0",
    reviewers = 1,
    convos    = false,
  required_tests = [
    "test (1.16.x, ubuntu-latest, amd64, 15.x)",
    "1.15",
    "1.15-el7"]
  },
]