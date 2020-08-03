data "terraform_remote_state" "infra" {
  backend = "remote"

  config = {
    organization = "Tyk"
    workspaces = {
      name = "dev-euc1"
    }
  }
}

output "vpc_id" {
  value = data.terraform_remote_state.infra.outputs.vpc_id
  description = "VPC in which the infra is running"
}
