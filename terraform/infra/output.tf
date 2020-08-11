data "terraform_remote_state" "infra" {
  backend = "remote"

  config = {
    organization = "Tyk"
    workspaces = {
      name = "infra-prod"
    }
  }
}

output "vpc_id" {
  value = data.terraform_remote_state.infra.outputs.vpc_id
  description = "VPC in which the infra is running"
}
