tyk_release_branches = [
  { branch    = "master",
    reviewers = 2,
    convos    = false,
  required_tests = [ 
    "Go 1.19.x Redis 5",
    "1.19-bullseye"]
  },
  { branch    = "release-4",
    reviewers = 0,
    convos    = false,
  required_tests = [ 
    "Go 1.16 Redis 5",
    "1.16",
    "1.16-el7"]
  },
  { branch        = "release-4.3",
    reviewers     = 0,
    convos        = false,
    source_branch = "release-4",
  required_tests = [ 
    "Go 1.16 Redis 5",
    "1.16",
    "1.16-el7"]
  },
  { branch        = "release-4.3.0",
    reviewers     = 0,
    convos        = false,
    source_branch = "release-4",
  required_tests = [ 
    "Go 1.16 Redis 5",
    "1.16",
    "1.16-el7"]
  },
  { branch        = "release-4.3.1",
    reviewers     = 0,
    convos        = false,
    source_branch = "release-4.3",
  required_tests = [ 
    "Go 1.16 Redis 5",
    "1.16",
    "1.16-el7"]
  },
  { branch        = "release-4.3.2",
    reviewers     = 0,
    convos        = false,
    source_branch = "release-4.3",
  required_tests = [ 
    "Go 1.16 Redis 5",
    "1.16",
    "1.16-el7"]
  },
  { branch        = "release-4.0.10",
    reviewers     = 0,
    convos        = false,
    source_branch = "release-4-lts",
  required_tests = [ 
    "Go 1.15 Redis 5",
    "1.15",
    "1.15-el7"]
  },
  { branch        = "release-4.0.11",
    reviewers     = 0,
    convos        = false,
    source_branch = "release-4-lts",
  required_tests = [ 
    "Go 1.15 Redis 5",
    "1.15",
    "1.15-el7"]
  }
]  