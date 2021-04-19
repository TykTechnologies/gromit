variable "base" {
  description = "State to use for base resources"
  type = string
}

variable "infra" {
  description = "State to use for infra resources"
  type = string
}

variable "name" {
  description = "The DNS record will be name-{gw,db,etc}"
  type = string
}

variable "tyk" {
  description = "Image tag for the tyk service"
  type        = string
}

variable "tyk-analytics" {
  description = "Image tag for the tyk-analytics service"
  type        = string
}

variable "tyk-pump" {
  description = "Image tag for the tyk-pump service"
  type        = string
}
