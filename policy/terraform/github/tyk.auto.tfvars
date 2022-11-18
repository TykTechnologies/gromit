tyk_release_branches = [
  { branch    = "master",
    reviewers = 2,
    convos    = false,
  	required_tests = ["Go 1.16 Redis 5"] },
  { branch    = "release-3-lts",
    reviewers = 2,
    convos    = false,
  	required_tests = ["Go 1.16 Redis 5"] },
  { branch    = "release-4",
    reviewers = 2,
    convos    = false,
  	required_tests = ["Go 1.16 Redis 5"] },
] 