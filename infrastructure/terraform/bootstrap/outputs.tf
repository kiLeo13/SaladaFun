output "bucket" {
  description = "Generated state bucket name to place in backend.hcl."
  value       = oci_objectstorage_bucket.terraform_state.name
}

output "namespace" {
  description = "Object Storage namespace to place in backend.hcl."
  value       = data.oci_objectstorage_namespace.current.namespace
}
