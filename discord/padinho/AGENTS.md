# Padinho contribution guidance

- Keep the Go baseline at 1.26 and preserve static builds with `CGO_ENABLED=0`.
- Keep command declaration, validation, options, and dispatch independent of
  GORM and OCI types. The interaction-bound responder deliberately accepts a
  native DiscordGo response; do not duplicate DiscordGo's response/component
  model in project-owned structs.
- Register every command through the single composition root in
  `internal/commands`. `discord.Routes` owns separate application-command and
  literal message-command registries because their transport contracts differ;
  freeze and dispatch both through the same routes and gateway lifecycle.
  Feature registration functions may delegate from that composition root; do
  not create an independent routes instance or command lifecycle.
- Register stable component and modal routes with the same `discord.Routes`
  composition. Keep reconstructible state in validated `custom_id` parameters;
  add expiring server-side state only for flows that cannot be reconstructed.
- Accept handlers by function signature. A feature may use standalone functions
  or bound methods and may group several handlers in one file when cohesive.
- Put command names, descriptions, options, and middleware beside the feature's
  registration code. Keep handlers focused on execution.
- Reserve `context.Context` for cancellation and deadlines. Add explicit typed
  fields to `command.CommandRequest` for application data.
- Preserve middleware order: registry, command group, subcommand group, route,
  then handler. Middleware may reject without calling `next`.
- Add tests for every behavior change. Critical registry and dispatch paths must
  retain 100% statement coverage; database changes require live MySQL tests.
- Do not expose HTTP ports. The bot communicates outbound to Discord and over the
  private subnet to MySQL.
- Give Padinho a dedicated DML-only MySQL account. Never put the MySQL
  administrator credential in Padinho's environment or Ansible variables; it is
  reserved for Terraform provisioning and interactive migration maintenance.
- Keep the runtime database password in the ignored, encrypted Ansible Vault
  host-variable file. Do not reintroduce OCI Vault/KMS for this deployment or
  commit decrypted production variables.
- Keep runtime images distroless, non-root, read-only, and within the configured
  384 MB container memory limit.
- Keep Padinho's image and GitHub Actions workflow independent of the root
  database module. Database migrations are built and applied manually, never by
  Padinho's image, Compose, or CI.
- Publish only `ghcr.io/kileo13/salada:latest` from Padinho's `master` branch;
  pull requests may validate the image without publishing it.
- Update this project's README and architecture documentation with every runtime,
  command framework, persistence, or deployment change.
- Keep user-facing text in `internal/locale/ptbr`; command names are the only
  user-facing identifiers that may remain untranslated.
- Organize growing features through `internal/domain/entity`,
  `internal/application/<feature>`, and concrete adapters such as
  `internal/persistence/mysql`. Discord handlers and jobs consume application
  services rather than querying repositories directly.
- Declare small interfaces in the package that consumes them. Keep concrete
  GORM repositories in `internal/persistence/mysql`, and keep test fakes beside
  their consumers unless substantial reuse justifies a shared test utility.
- Group cohesive slash commands, buttons, modals, and feature-specific Discord
  listeners in one feature package. Do not create one directory per handler.
  Keep process-scheduled work under `internal/job`.
- Keep `cmd/padinho` as explicit dependency wiring. Name repositories, services,
  senders, jobs, and other reusable dependencies before passing them onward;
  avoid nested constructor chains. Separate large composition roots with short
  section comments such as repositories, services, Discord, and scheduled jobs.
- Avoid blank-identifier compile-time interface assertions when ordinary
  constructor wiring already verifies the implementation. Use one only when it
  provides otherwise-missing compile-time coverage and explain why.
- Split deeply nested payloads and composite literals into meaningful locals or
  focused helpers when that makes the data flow easier to scan. Prefer named
  constants over unexplained limits and magic values, but do not extract small
  cohesive expressions merely to manufacture abstraction.
- Bootstrap MySQL directly from the individual `DB_*` environment variables and
  return `*gorm.DB`. Do not introduce a general environment `Config`, a
  `DBConfig`, or a caller-provided DSN. Schema history and migration execution
  remain solely owned by the root `database` module.
