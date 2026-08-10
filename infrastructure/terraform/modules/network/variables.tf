variable "compartment_ocid" { type = string }
variable "name" { type = string }
variable "vcn_cidr" { type = string }
variable "bot_subnet_cidr" { type = string }
variable "database_subnet_cidr" { type = string }
variable "admin_cidr" {
  type = string
  validation {
    condition     = can(cidrhost(var.admin_cidr, 0)) && endswith(var.admin_cidr, "/32")
    error_message = "admin_cidr must be the administrator's single IPv4 address in /32 notation."
  }
}
