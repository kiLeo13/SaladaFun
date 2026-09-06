# Discord Utils contribution guidance

- Keep one Discord gateway lifecycle owner for this mod. Features must not
  construct their own JDA clients.
- Treat bot tokens and webhook URLs as secrets: redact them from diagnostics,
  documentation examples, and test output.
- Keep Minecraft/NeoForge adapters separate from Discord transport code.
- Discord account linking is intentionally outside this project's initial scope.
