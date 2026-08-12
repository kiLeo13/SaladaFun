# SaladaFun

SaladaFun is a monorepo for the applications and integrations used by a private
Discord guild and its game servers. Projects share repository-wide standards
while remaining independently buildable and deployable.

## Repository layout

```text
.
|-- database/          Shared SQL migrations
|-- infrastructure/    OCI Terraform and host Ansible
|-- discord/
|   `-- padinho/       Padinho Discord bot
|-- minecraft/
|   `-- salada/        Java 25/Purpur 26.2 Minecraft plugin
|-- AGENTS.md          Repository-wide contribution guidance
`-- ARCHITECTURE.md    Ecosystem structure and project index
```

## Projects

| Project | Description | Documentation |
| --- | --- | --- |
| Salada | Minecraft gameplay features and an optional Discord chat bridge | [`minecraft/salada`](minecraft/salada/README.md) |
| Padinho | Typed Discord bot foundation for the guild | [`discord/padinho`](discord/padinho/README.md) |

Infrastructure and migrations are shared operational concerns. Their runbooks
live in [`infrastructure/terraform`](infrastructure/terraform/README.md),
[`infrastructure/ansible`](infrastructure/ansible/README.md), and
[`database`](database/README.md).

## Verification

Build Salada with JDK 25 and Maven:

```text
mvn -f minecraft/salada/pom.xml clean package
```

Verify Padinho with Go 1.26:

```text
cd discord/padinho
go test -race ./...
```

Verify the independent migration project with the same Go baseline:

```text
cd database
go test -race ./...
```

## License

This repository does not currently include a license. Add an explicit license
before distributing or accepting contributions under defined reuse terms.
