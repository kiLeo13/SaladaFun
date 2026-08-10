output "runtime_secret_ocid" { value = oci_vault_secret.runtime_environment.id }
output "registry_secret_ocid" { value = oci_vault_secret.registry_credentials.id }
