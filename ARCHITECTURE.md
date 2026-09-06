# SaladaFun ecosystem architecture

## Purpose

SaladaFun is a monorepo for independently deployable applications used by one
Discord guild and its game servers. Projects share ownership and conventions
without being forced into the same language or build system.

## Repository structure

```text
.
|-- database/                 Ecosystem SQL migration history
|-- infrastructure/
|   |-- ansible/              VM configuration and Compose reconciliation
|   `-- terraform/            OCI bootstrap, modules, and production stack
|-- discord/
|   `-- padinho/              Padinho Discord bot
|-- minecraft/
|   |-- neoforge/
|   |   `-- discord-utils/    NeoForge Discord chat bridge
|   `-- purpur/
|       `-- saladafun/        SaladaFun Purpur plugin
|-- AGENTS.md
|-- ARCHITECTURE.md
`-- README.md
```

The root owns ecosystem governance, navigation, architecture, shared schema,
and infrastructure. Each deployable project owns its implementation, tests,
build, operational documentation, and detailed architecture. There is no root
language-specific aggregator.

Shared migrations are a cross-release contract owned by the independent root
`database` Go module. Its self-contained Linux executable is built and run
manually on the Padinho VM; application code and Compose never own schema
evolution.

The shared `discord_account_links` table stores only Discord accounts that
participate in a hierarchy. It uses an auto-increment surrogate key plus a
unique Discord snowflake and optional direct-parent snowflake, allowing arbitrary
rooted trees such as a managed account with its own managed accounts. Missing
parents and direct self-parenting are rejected by MySQL. Applications that add
reparenting must reject longer cycles before updating a relationship; hierarchy
traversal and command behavior remain application concerns.
Terraform owns OCI resources; Ansible configures the resulting VM and
reconciles Docker Compose. GitHub Actions tests and publishes Padinho's image to
the public GitHub Container Registry package used by Compose.
The private MySQL subnet admits TCP/3306 from the bot subnet through a subnet
security list. MySQL deliberately has no NSG attachment because OCI requires
separate `mysqldbsystem` resource-principal IAM policies to provision one.

## Projects

### Minecraft: SaladaFun

`minecraft/purpur/saladafun` is a Java 25/Purpur 26.2 plugin providing shared player
vitals, batch breaking, and an optional Discord chat bridge. See
[`minecraft/purpur/saladafun/ARCHITECTURE.md`](minecraft/purpur/saladafun/ARCHITECTURE.md).

### Minecraft: Discord Utils

`minecraft/neoforge/discord-utils` is a Java 21/NeoForge 21.1.249 server-side
Discord utility mod. Its first feature is the optional bidirectional chat bridge.

### Discord: Padinho

Padinho, under `discord/padinho`, is a Go 1.26 application designed for a 1 GB
`VM.Standard.E2.1.Micro`. A frozen route composition owns slash commands,
stateless message components, and modal submissions. Command declarations stay
typed while responders deliberately accept native DiscordGo payloads so Salada
does not maintain a second copy of the evolving Discord component model.
Application data lives in typed requests rather than context values.

The first feature stores birthdays in MySQL, presents one Components V2 page
per month, accepts registrations through a modal, and checks local calendar
dates once per minute. A delivery ledger makes repeated scheduler checks
idempotent. The composition root opens GORM and retrieves the Discord token and
birthday channel through the database-backed `config` repository before
opening the gateway. Discord derives its application identity from the
connected bot and synchronizes global commands unconditionally. See
[`discord/padinho/ARCHITECTURE.md`](discord/padinho/ARCHITECTURE.md).

## Verification

Projects remain independent:

```text
mvn -f minecraft/purpur/saladafun/pom.xml clean package
cd minecraft/neoforge/discord-utils && .\gradlew.bat clean build
cd discord/padinho && go test -race ./...
```
