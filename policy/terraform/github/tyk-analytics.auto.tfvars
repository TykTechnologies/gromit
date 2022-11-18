tyk-analytics_release_branches = [
  { branch    = "master",
    reviewers = 0,
    convos    = false,
    required_tests = ["commit message linter", "test (1.16.x, ubuntu-latest, amd64, 15.x)", "sqlite", "ci", "mongo"] },
  { branch    = "release-3-lts",
    reviewers = 0,
    convos    = false,
    required_tests = ["commit message linter", "test (1.16.x, ubuntu-latest, amd64, 15.x)", "sqlite", "ci", "mongo"] },
  { branch    = "release-4",
    reviewers = 0,
    convos    = false,
    required_tests = ["commit message linter", "test (1.16.x, ubuntu-latest, amd64, 15.x)", "sqlite", "ci", "mongo"] },
]