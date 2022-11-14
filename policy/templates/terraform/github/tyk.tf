module "tyk" {
  source               = "../../../modules/github-repos"
  repo                 = "{{ .Name }}"
  description          = "Tyk Open Source API Gateway written in Go, supporting REST, GraphQL, TCP and gRPC protocols"
  topics               = ["api", "api-gateway", "api-management", "cloudnative", "go", "graphql", "grpc", "k8s", "kubernetes", "microservices", "reverse-proxy", "tyk"]
  wiki                 = false
  default_branch       = "master"
  vulnerability_alerts = true
  release_branches = [
    { branch         = "master",
      reviewers      = 2,
      required_tests = ["Go 1.16 Redis 5"],
    convos = false },
    { branch         = "release-4.3",
      reviewers      = 0,
      source_branch  = "release-4",
      required_tests = ["Go 1.16 Redis 5"],
    convos = false },
    { branch         = "release-4.3.0",
      reviewers      = 0,
      source_branch  = "release-4",
      required_tests = ["Go 1.16 Redis 5"],
    convos = false },
  ]
}