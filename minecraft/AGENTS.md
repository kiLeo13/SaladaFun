# Minecraft contribution guidance

These rules apply to every project under `minecraft/` and supplement the root
repository guidance.

- Put loader-wide contribution guidance in the loader directory and keep
  project-specific rules beside the project.
- Keep platform APIs out of reusable domain code. Perform loader adaptation at
  the project platform boundary.
- Keep plugin descriptors, default configuration, and bundled migrations in the
  standard project resources directory.
- Document project-specific deployment artifacts and live-server verification in
  that project's documentation.
