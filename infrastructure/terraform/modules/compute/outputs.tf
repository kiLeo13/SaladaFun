output "instance_id" {
  description = "OCID of the Padinho compute instance."
  value       = oci_core_instance.bot.id
}

output "public_ip" {
  description = "Public IPv4 address used for restricted SSH access."
  value       = oci_core_instance.bot.public_ip
}

output "private_ip" {
  description = "Private IPv4 address of the Padinho instance."
  value       = oci_core_instance.bot.private_ip
}
