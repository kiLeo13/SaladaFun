output "bot_subnet_id" {
  description = "OCID of Padinho's public subnet."
  value       = oci_core_subnet.bot.id
}

output "database_subnet_id" {
  description = "OCID of MySQL's private subnet."
  value       = oci_core_subnet.database.id
}

output "bot_nsg_id" {
  description = "OCID of Padinho's network security group."
  value       = oci_core_network_security_group.bot.id
}

output "database_nsg_id" {
  description = "OCID of MySQL's network security group."
  value       = oci_core_network_security_group.database.id
}
