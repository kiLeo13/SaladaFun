variable "compartment_ocid" {
  description = "Compartment that owns the Terraform state bucket."
  type        = string
}

variable "region" {
  description = "OCI home region for the state bucket."
  type        = string
}

variable "state_bucket_name" {
  description = "Globally unique name within the tenancy Object Storage namespace."
  type        = string
  default     = "saladafun-terraform-state"
}
