variable "tenancy_ocid" { type = string }
variable "compartment_ocid" { type = string }
variable "region" { type = string }
variable "availability_domain" { type = string }
variable "admin_cidr" { type = string }
variable "ssh_public_key" {
  type      = string
  sensitive = true
}
variable "discord_token" {
  type      = string
  sensitive = true
  validation {
    condition     = length(var.discord_token) > 0 && !strcontains(var.discord_token, "\n")
    error_message = "discord_token must be non-empty and contain no newlines."
  }
}
variable "discord_application_id" { type = string }
variable "discord_guild_id" {
  description = "Development guild for immediate command updates; empty means global commands."
  type        = string
  default     = ""
}
variable "ghcr_username" { type = string }
variable "ghcr_token" {
  description = "Fine-grained GitHub token limited to read:packages."
  type        = string
  sensitive   = true
}
variable "mysql_admin_username" {
  type    = string
  default = "padinho_admin"
}
variable "mysql_admin_password" {
  type      = string
  sensitive = true
  validation {
    condition     = length(var.mysql_admin_password) >= 8 && length(var.mysql_admin_password) <= 32 && can(regex("^[A-Za-z0-9!#$%&*+_=?.-]+$", var.mysql_admin_password))
    error_message = "mysql_admin_password must be 8-32 characters and use only DSN-safe letters, digits, or !#$%&*+_=?.-."
  }
}
variable "mysql_database_name" {
  type    = string
  default = "padinho"
  validation {
    condition     = can(regex("^[a-z][a-z0-9_]*$", var.mysql_database_name))
    error_message = "mysql_database_name must be a lowercase MySQL identifier."
  }
}
