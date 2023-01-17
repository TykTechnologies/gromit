portal_release_branches = [
  { branch    = "master",
    reviewers = 1,
    convos    = false,
  required_tests = [
    "test (1.16.x, ubuntu-latest, amd64, 15.x)",
    "1.15",
    "1.15-el7",
    "ci",
    "upgrade-deb (amd64, ubuntu:xenial)",
    "upgrade-deb (amd64, ubuntu:bionic)",
    "upgrade-deb (amd64, ubuntu:focal)",
    "upgrade-deb (amd64, debian:bullseye)",
    "upgrade-deb (arm64, ubuntu:xenial)",
    "upgrade-deb (arm64, ubuntu:bionic)",
    "upgrade-deb (arm64, ubuntu:focal)",
    "upgrade-deb (arm64, debian:bullseye)",
    "upgrade-rpm (ubi7/ubi)",
    "upgrade-rpm (ubi8/ubi)",
    "smoke-tests"]
  },
]