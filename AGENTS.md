# SaladaFun ecosystem contribution guidance

This file applies to the entire repository. More specific `AGENTS.md` files
under platform and project directories add rules for their scope.

- Keep deployable applications grouped first by platform and then by project,
  for example `minecraft/salada`.
- Keep each project independently buildable, testable, deployable, and
  documented. Do not couple unrelated projects through implicit working-directory
  assumptions.
- Keep only repository-wide configuration, governance, architecture, licensing,
  and navigation files at the root alongside the platform directories.
- Put implementation documentation beside the project it describes. Update the
  root `ARCHITECTURE.md` when the ecosystem structure or project boundaries
  change.
- Add or update tests for every behavior change and run the complete build for
  each affected project before considering work complete.
- Never commit generated build output, runtime state, secrets, or IDE user
  settings.
- Do not perform Git operations that affect a remote unless explicitly asked.
- Do not perform destructive operations.
