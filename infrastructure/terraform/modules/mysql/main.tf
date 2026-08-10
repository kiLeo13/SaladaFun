resource "oci_mysql_mysql_db_system" "this" {
  availability_domain     = var.availability_domain
  compartment_id          = var.compartment_ocid
  subnet_id               = var.subnet_id
  nsg_ids                 = var.nsg_ids
  display_name            = var.name
  hostname_label          = "saladadb"
  shape_name              = var.database_shape
  admin_username          = var.admin_username
  admin_password          = var.admin_password
  data_storage_size_in_gb = 50
  is_highly_available     = false

  backup_policy {
    is_enabled = false
  }

  deletion_policy {
    automatic_backup_retention = "DELETE"
    final_backup               = "SKIP_FINAL_BACKUP"
    is_delete_protected        = true
  }
}

resource "oci_mysql_heat_wave_cluster" "this" {
  db_system_id = oci_mysql_mysql_db_system.this.id
  shape_name   = var.heatwave_shape
  cluster_size = 1
}
