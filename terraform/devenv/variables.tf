variable "name_prefix" {
  description = "The DNS record will be name_prefix-{gw,db,etc}"
  type = string
}

variable "vpc_id" {
  description = "VPC that base and infra are on"
  type = string
}

variable "tyk" {
  description = "Image for the tyk service"
  type        = string
}

variable "tyk-analytics" {
  description = "Image for the tyk-analytics service"
  type        = string
}

variable "tyk-pump" {
  description = "Image for the tyk-pump service"
  type        = string
}

variable "region" {
  type = string
}

variable "config_efs" {
  description = "EFS volume with tyk configurations"
  type        = string
}

variable "cfssl_efs" {
  description = "EFS volume with CFSSL keys and certs"
}
