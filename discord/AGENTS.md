# Discord contribution guidance

These rules apply to every project under `discord/` and supplement the root
repository guidance.

- Keep each Discord application in its own direct project directory with an
  independent build, tests, README, and architecture document.
- Prefer static, strongly typed implementations that fit the 1 GB OCI compute
  budget; the current Go baseline is 1.26.
- Keep Discord library types at adapter boundaries. Domain and command code must
  depend on project-owned types and interfaces.
- Register application commands through one composition root per application.
- Reserve `context.Context` for cancellation and deadlines rather than untyped
  application data.
- Do not expose inbound application ports unless an architecture change
  explicitly requires and documents them.
- Run the affected application's full test suite after changes. Persistence
  changes require integration tests against a live database.
