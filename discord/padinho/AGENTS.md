# Padinho contribution guidance

- Keep the Go baseline at 1.26 and preserve static builds with `CGO_ENABLED=0`.
- Keep `internal/command` independent of DiscordGo, GORM, and OCI types.
- Register every command through the single composition root in
  `internal/commands`. Feature registration functions may delegate from there;
  do not create a second registry.
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
- Keep runtime images distroless, non-root, read-only, and within the configured
  384 MB container memory limit.
- Update this project's README and architecture documentation with every runtime,
  command framework, persistence, or deployment change.
