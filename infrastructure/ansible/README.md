# Ansible

Ansible configures the existing Terraform VM; it does not provision OCI
resources. It installs Docker Engine and Compose from Docker's signed Ubuntu
repository, bounds container logs for the 1 GB host, writes Padinho's restricted
database environment from Ansible variables, and reconciles the Compose project.
Compose pulls Salada's public GHCR image anonymously and starts Padinho without
applying database migrations. Build, upload, and run the root database migration
executable manually with the separate MySQL administrator account before
deploying code that requires new schema.

```sh
cd infrastructure/ansible
ansible-galaxy collection install -r requirements.yml
cp inventories/production.yml.example inventories/production.yml
cp host_vars/salada/vault.yml.example host_vars/salada/vault.yml
# Fill ansible_host from `terraform output -raw bot_public_ip` and
# padinho_database_host/name from the matching MySQL Terraform outputs.
# Put only Padinho's restricted database password in vault.yml, then encrypt it.
ansible-vault encrypt host_vars/salada/vault.yml
ansible-playbook --ask-vault-pass site.yml
ansible-playbook --ask-vault-pass site.yml  # Expected: no changes.
```

The ignored inventory contains connection details and non-secret runtime
settings. The ignored Ansible Vault file contains only
`vault_padinho_database_password`; its password is requested locally for every
playbook run. Ansible decrypts it in memory and writes `/etc/salada/salada.env`
as root-owned mode `0600`. The MySQL administrator password never enters
Ansible or Padinho.

SSH remains the only inbound host port and OCI limits it to the configured
administrator `/32`. Padinho and MySQL communicate privately. Watchtower polls
GHCR every 300 seconds and only updates containers carrying Salada's explicit
enable/scope labels. Its archived status is an accepted project constraint, so
the image is digest-pinned, its HTTP API is not enabled, and Compose pins its
Docker client API to `1.44` for compatibility with the installed daemon.
