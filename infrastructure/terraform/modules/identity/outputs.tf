output "runtime_secret_ocid" {
  description = "OCID of the Vault secret containing Padinho's environment."
  value       = oci_vault_secret.runtime_environment.id
}
