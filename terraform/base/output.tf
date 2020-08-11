data "terraform_remote_state" "integration" {
  backend = "remote"

  config = {
    organization = "Tyk"
    workspaces = {
      name = "base-prod"
    }
  }
}

# repo_rurls should probably be generated from env.Repos
output "repo_urls" {
  value = map("tyk", data.terraform_remote_state.integration.outputs.tyk["ecr"],
    "tyk-analytics", data.terraform_remote_state.integration.outputs.tyk-analytics["ecr"],
    "tyk-pump", data.terraform_remote_state.integration.outputs.tyk-pump["ecr"])
  description = "ECR base URLs"
}

output "region" {
  value = data.terraform_remote_state.integration.outputs.region
  description = "Region in which the env is running"
}

output "cfssl_efs" {
  value = data.terraform_remote_state.integration.outputs.cfssl_efs
  description = "EFS for keys and certs"
}

output "config_efs" {
  value = data.terraform_remote_state.integration.outputs.config_efs
  description = "EFS for Tyk config"
}
