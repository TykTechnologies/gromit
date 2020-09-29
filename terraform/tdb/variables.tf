variable "base" {
  description = "State to use for base resources"
  type        = string
}

variable "infra" {
  description = "State to use for infra resources"
  type        = string
}

variable "name" {
  description = "The DNS record will be {env_name}-{name}.GROMIT_DOMAIN"
  type        = string
}

variable "env_name" {
  description = "The environment in which to spin up the pod"
  type        = string
}

variable "image" {
  description = "Fully qualified image ref to run in the pod"
  type        = string
}

variable "rep_count" {
  description = "Number of instances to run, implemented by count."
  type        = number
  default     = 1
}

variable "cmdline" {
  description = "Optional command line args to supply to the run"
  type        = list
  default     = []
}
