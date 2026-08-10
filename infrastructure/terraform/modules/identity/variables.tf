variable "tenancy_ocid" { type = string }
variable "compartment_ocid" { type = string }
variable "instance_id" { type = string }
variable "name" { type = string }
variable "runtime_environment" {
  type      = string
  sensitive = true
}
variable "registry_credentials" {
  type      = string
  sensitive = true
}
