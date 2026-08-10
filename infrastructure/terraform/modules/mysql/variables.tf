variable "compartment_ocid" { type = string }
variable "availability_domain" { type = string }
variable "subnet_id" { type = string }
variable "nsg_ids" { type = list(string) }
variable "name" { type = string }
variable "admin_username" { type = string }
variable "admin_password" {
  type      = string
  sensitive = true
}
variable "database_shape" {
  type    = string
  default = "MySQL.Free"
  validation {
    condition     = var.database_shape == "MySQL.Free"
    error_message = "The database shape must remain MySQL.Free."
  }
}
variable "heatwave_shape" {
  type    = string
  default = "HeatWave.Free"
  validation {
    condition     = var.heatwave_shape == "HeatWave.Free"
    error_message = "The HeatWave shape must remain HeatWave.Free."
  }
}
