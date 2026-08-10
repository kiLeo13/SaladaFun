output "bot_subnet_id" { value = oci_core_subnet.bot.id }
output "database_subnet_id" { value = oci_core_subnet.database.id }
output "bot_nsg_id" { value = oci_core_network_security_group.bot.id }
output "database_nsg_id" { value = oci_core_network_security_group.database.id }
