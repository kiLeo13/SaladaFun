mock_provider "oci" {}

variables {
  compartment_ocid     = "ocid1.compartment.oc1..example"
  name                 = "salada-test"
  vcn_cidr             = "10.42.0.0/16"
  bot_subnet_cidr      = "10.42.10.0/24"
  database_subnet_cidr = "10.42.20.0/24"
  admin_cidr           = "203.0.113.10/32"
}

run "database_uses_subnet_security_list" {
  command = apply

  assert {
    condition = (
      length(oci_core_subnet.database.security_list_ids) == 1 &&
      contains(oci_core_subnet.database.security_list_ids, oci_core_security_list.database.id)
    )
    error_message = "The database subnet must use the managed database security list."
  }
}

run "mysql_is_reachable_only_from_bot_subnet" {
  command = plan

  assert {
    condition = (
      length(oci_core_security_list.database.ingress_security_rules) == 1 &&
      one(oci_core_security_list.database.ingress_security_rules).protocol == "6" &&
      one(oci_core_security_list.database.ingress_security_rules).source == var.bot_subnet_cidr &&
      one(one(oci_core_security_list.database.ingress_security_rules).tcp_options).min == 3306 &&
      one(one(oci_core_security_list.database.ingress_security_rules).tcp_options).max == 3306
    )
    error_message = "The database security list must allow only TCP/3306 from the bot subnet."
  }
}

run "database_retains_outbound_access" {
  command = plan

  assert {
    condition = (
      length(oci_core_security_list.database.egress_security_rules) == 1 &&
      one(oci_core_security_list.database.egress_security_rules).protocol == "all" &&
      one(oci_core_security_list.database.egress_security_rules).destination == "0.0.0.0/0"
    )
    error_message = "The database security list must retain outbound access."
  }
}

run "rejects_non_host_admin_cidr" {
  command = plan

  variables {
    admin_cidr = "203.0.113.0/24"
  }

  expect_failures = [var.admin_cidr]
}
