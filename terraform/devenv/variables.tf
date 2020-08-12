variable "base" {
  description = "State to use for base resources"
  type = string
}

variable "infra" {
  description = "State to use for infra resources"
  type = string
}

variable "name_prefix" {
  description = "The DNS record will be name_prefix-{gw,db,etc}"
  type = string
}

variable "tyk_tag" {
  description = "Image tag for the tyk service"
  type        = string
}

variable "tyk-analytics_tag" {
  description = "Image tag for the tyk-analytics service"
  type        = string
}

variable "tyk-pump_tag" {
  description = "Image tag for the tyk-pump service"
  type        = string
}
