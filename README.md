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
|   |-- neoforge/
|   |   `-- discord-utils/ Java 21/NeoForge 21.1.249 Discord utilities
|   `-- purpur/
|       `-- saladafun/ Java 25/Purpur 26.2 Minecraft plugin
|-- AGENTS.md          Repository-wide contribution guidance
`-- ARCHITECTURE.md    Ecosystem structure and project index
```

## Projects

| Project | Description | Documentation |
| --- | --- | --- |
| SaladaFun | Minecraft gameplay features and an optional Discord chat bridge | [`minecraft/purpur/saladafun`](minecraft/purpur/saladafun/README.md) |
| Discord Utils | NeoForge Discord chat bridge | [`minecraft/neoforge/discord-utils`](minecraft/neoforge/discord-utils/README.md) |
| Padinho | Typed Discord bot foundation for the guild | [`discord/padinho`](discord/padinho/README.md) |

Infrastructure and migrations are shared operational concerns. Their runbooks
live in [`infrastructure/terraform`](infrastructure/terraform/README.md),
[`infrastructure/ansible`](infrastructure/ansible/README.md), and
[`database`](database/README.md).

## Verification

Build SaladaFun with JDK 25 and Maven:

```text
mvn -f minecraft/purpur/saladafun/pom.xml clean package
```

Verify Padinho with Go 1.26:

```text
cd discord/padinho
go test -race ./...
```

Build and test Discord Utils with JDK 21:

```text
cd minecraft/neoforge/discord-utils
.\gradlew.bat clean build
```

Verify the independent migration project with the same Go baseline:

```text
cd database
go test -race ./...
```
