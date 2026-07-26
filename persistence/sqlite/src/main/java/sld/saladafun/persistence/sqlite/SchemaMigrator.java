package sld.saladafun.persistence.sqlite;

import org.jooq.DSLContext;

import java.io.IOException;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.time.Instant;
import java.util.HexFormat;

/**
 * Applies checksum-protected, versioned SQL migrations.
 */
final class SchemaMigrator {
    private static final int VERSION = 1;
    private static final String RESOURCE = "/db/migration/V001__initial_schema.sql";

    void migrate(DSLContext context) {
        String script = readScript();
        String checksum = checksum(script);
        boolean historyExists = context.fetchOne(
            "SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = 'schema_history'"
        ) != null;
        if (historyExists) {
            var existing = context.fetchOne(
                "SELECT checksum FROM schema_history WHERE version = ?", VERSION
            );
            if (existing != null) {
                String installedChecksum = existing.get("checksum", String.class);
                if (!checksum.equals(installedChecksum)) {
                    throw new PersistenceException("Checksum mismatch for schema migration V001");
                }
                return;
            }
        }

        context.transaction(configuration -> {
            DSLContext transaction = configuration.dsl();
            for (var query : transaction.parser().parse(script)) {
                query.execute();
            }
            transaction.execute(
                "INSERT INTO schema_history(version, description, checksum, installed_at) "
                    + "VALUES (?, ?, ?, ?)",
                VERSION,
                "initial_schema",
                checksum,
                Instant.now().toString()
            );
        });
    }

    private String readScript() {
        try (InputStream stream = SchemaMigrator.class.getResourceAsStream(RESOURCE)) {
            if (stream == null) {
                throw new PersistenceException("Missing schema migration " + RESOURCE);
            }
            return new String(stream.readAllBytes(), StandardCharsets.UTF_8);
        } catch (IOException exception) {
            throw new PersistenceException("Could not read schema migration " + RESOURCE, exception);
        }
    }

    private String checksum(String script) {
        try {
            return HexFormat.of().formatHex(
                MessageDigest.getInstance("SHA-256")
                    .digest(script.getBytes(StandardCharsets.UTF_8))
            );
        } catch (NoSuchAlgorithmException impossible) {
            throw new IllegalStateException("SHA-256 is required by the Java platform", impossible);
        }
    }
}
