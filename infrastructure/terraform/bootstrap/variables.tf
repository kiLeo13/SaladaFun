variable "compartment_ocid" {
  description = "Compartment that owns the Terraform state bucket."
  type        = string
}

variable "region" {
  description = "OCI region where the state bucket is created."
  type        = string
}
