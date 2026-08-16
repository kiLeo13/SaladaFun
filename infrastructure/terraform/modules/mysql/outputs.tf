output "db_system_id" {
  description = "OCID of the MySQL DB system."
  value       = oci_mysql_mysql_db_system.this.id
}

output "private_ip" {
  description = "Private IPv4 address of the MySQL DB system."
  value       = oci_mysql_mysql_db_system.this.ip_address
}
