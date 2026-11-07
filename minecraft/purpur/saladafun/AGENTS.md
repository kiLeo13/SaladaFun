# SaladaFun plugin contribution guidance

- Keep this project as one Maven JAR module using the standard project `src/main`
  and `src/test` layout.
- Keep the domain packages under `sld.saladafun.shared` and
  `sld.saladafun.batchbreaking` free of Bukkit, Paper, Purpur, JDBC, jOOQ, and
  SQLite types.
- Keep SQLite/jOOQ implementations under
  `sld.saladafun.persistence.sqlite` and server adapters under
  `sld.saladafun.platform.purpur`.
- Use the build commands in [README.md](README.md#build-and-test).
- Deploy only `target/saladafun-1.0.jar`; the `original-` JAR is an unshaded
  intermediate artifact.
