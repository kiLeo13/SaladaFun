# Minecraft contribution guidance

These rules apply to every project under `minecraft/` and supplement the root
repository guidance.

- Group Minecraft projects first by loader and then by independently deployable
  project, using `minecraft/<loader>/<project>`.
- Keep each project independently buildable and give it its own build file,
  source tree, tests, README, and architecture documentation.
- Put loader-wide contribution guidance in the loader directory and keep
  project-specific rules beside the project.
- Keep platform APIs out of reusable domain code. Perform loader adaptation at
  the project platform boundary.
- Keep plugin descriptors, default configuration, and bundled migrations in the
  standard project resources directory.
- Treat server databases, WAL/SHM companions, logs, and generated build
  directories as runtime or build output; never commit them.
- Run the affected project's complete build after changes. Database interface or
  persistence changes require an integration test against a real database.
- Document project-specific deployment artifacts and live-server verification in
  that project's documentation.
