# Ansible

Ansible configures the existing Terraform VM; it does not provision OCI
resources. It installs Docker Engine and Compose from Docker's signed Ubuntu
repository, bounds container logs for the 1 GB host, installs a pinned OCI CLI,
retrieves runtime and GHCR secrets through the instance principal, and
reconciles the Compose project.

```sh
cd infrastructure/ansible
ansible-galaxy collection install -r requirements.yml
cp inventories/production.yml.example inventories/production.yml
# Fill the public IP and the two sensitive Terraform outputs.
ansible-playbook site.yml --check --diff
ansible-playbook site.yml
ansible-playbook site.yml  # Expected: no changes (idempotence check).
```

SSH remains the only inbound host port and OCI limits it to the configured
administrator `/32`. The bot and MySQL communicate privately. Watchtower polls
GHCR every 300 seconds and only updates containers carrying Padinho's explicit
enable/scope labels. Its archived status is an accepted project constraint, so
the image is digest-pinned and its HTTP API is not enabled.
