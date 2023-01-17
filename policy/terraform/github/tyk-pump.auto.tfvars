tyk-pump_release_branches = [
  { branch    = "master",
    reviewers = 2,
    convos    = false,
  required_tests = [
    "1.15",
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
  { branch    = "release-1.7",
    reviewers = 2,
    convos    = false,
  required_tests = [
    "1.15",
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
  }
]