tyk-analytics_release_branches = [
  { branch    = "master",
    reviewers = 2,
    convos    = false,
  required_tests = [
    "1.19-bullseye",
    "test (1.19.x, ubuntu-latest, amd64, 15.x)",
    "sqlite",
    "mongo"]
  },
  { branch    = "release-4",
    reviewers = 0,
    convos    = false,
  required_tests = [
    "1.16",
    "1.16-el7",
    "test (1.16.x, ubuntu-latest, amd64, 15.x)",
    "sqlite",
    "mongo"]
  },
  { branch        = "release-4.3",
    reviewers     = 0,
    convos        = false,
    source_branch = "release-4",
  required_tests = [
    "1.16",
    "1.16-el7",
    "test (1.16.x, ubuntu-latest, amd64, 15.x)",
    "sqlite",
    "mongo"]
  },
  { branch        = "release-4.3.0",
    reviewers     = 0,
    convos        = false,
    source_branch = "release-4",
  required_tests = [
    "1.16",
    "1.16-el7",
    "test (1.16.x, ubuntu-latest, amd64, 15.x)",
    "sqlite",
    "mongo"]
  },
  { branch        = "release-4.3.1",
    reviewers     = 0,
    convos        = false,
    source_branch = "release-4.3",
  required_tests = [
    "1.16",
    "1.16-el7",
    "test (1.16.x, ubuntu-latest, amd64, 15.x)",
    "sqlite",
    "mongo"]
  },
  { branch        = "release-4.3.2",
    reviewers     = 0,
    convos        = false,
    source_branch = "release-4.3",
  required_tests = [
    "1.16",
    "1.16-el7",
    "test (1.16.x, ubuntu-latest, amd64, 15.x)",
    "sqlite",
    "mongo"]
  },
  { branch        = "release-4.0.10",
    reviewers     = 0,
    convos        = false,
    source_branch = "release-4-lts",
  required_tests = [
    "1.16",
    "1.16-el7",
    "test (1.16.x, ubuntu-latest, amd64, 15.x)",
    "sqlite",
    "mongo"]
  },
  { branch        = "release-4.0.11",
    reviewers     = 0,
    convos        = false,
    source_branch = "release-4-lts",
  required_tests = [
    "1.16",
    "1.16-el7",
    "test (1.16.x, ubuntu-latest, amd64, 15.x)",
    "sqlite",
    "mongo"]
  }
]
