# Minecraft contribution guidance

These rules apply to every project under `minecraft/` and supplement the root
repository guidance.

- Keep each Minecraft plugin or server-side application in its own direct
  project directory under `minecraft/`.
- Keep each project independently buildable and give it its own build file,
  source tree, tests, README, and architecture documentation.
- For Purpur plugins, keep the shared platform baseline at Java 25 and Purpur
  26.2 unless an explicitly documented ecosystem migration changes it.
- Scope server-provided APIs such as Purpur as `provided`; do not shade them into
  deployable plugin JARs.
- Keep platform APIs out of reusable domain code. Perform Bukkit, Paper, and
  Purpur adaptation at the project platform boundary.
- Keep plugin descriptors, default configuration, and bundled migrations in the
  standard project resources directory.
- Treat server databases, WAL/SHM companions, logs, and generated build
  directories as runtime or build output; never commit them.
- Run the affected project's complete build after changes. Database interface or
  persistence changes require an integration test against a real database.
- Document project-specific deployment artifacts and live-server verification in
  that project's documentation.
