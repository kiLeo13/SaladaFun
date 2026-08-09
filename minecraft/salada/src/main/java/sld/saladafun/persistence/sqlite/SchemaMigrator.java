package sld.saladafun.persistence.sqlite;

import org.jooq.DSLContext;

import java.io.IOException;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.time.Instant;
import java.util.HexFormat;
import java.util.List;

/**
 * Applies checksum-protected, versioned SQL migrations.
 */
final class SchemaMigrator {
    private static final List<Migration> MIGRATIONS = List.of(
        new Migration(1, "initial_schema", "/db/migration/V001__initial_schema.sql"),
        new Migration(2, "shared_effects", "/db/migration/V002__shared_effects.sql")
    );

    void migrate(DSLContext context) {
        for (Migration migration : MIGRATIONS) {
            migrate(context, migration);
        }
    }

    private void migrate(DSLContext context, Migration migration) {
        String script = readScript(migration.resource());
        String checksum = checksum(script);
        boolean historyExists = context.fetchOne(
            "SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = 'schema_history'"
        ) != null;
        if (historyExists) {
            var existing = context.fetchOne(
                "SELECT checksum FROM schema_history WHERE version = ?",
                migration.version()
            );
            if (existing != null) {
                String installedChecksum = existing.get("checksum", String.class);
                if (!checksum.equals(installedChecksum)) {
                    throw new PersistenceException(
                        "Checksum mismatch for schema migration V%03d".formatted(
                            migration.version()
                        )
                    );
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
                migration.version(),
                migration.description(),
                checksum,
                Instant.now().toString()
            );
        });
    }

    private String readScript(String resource) {
        try (InputStream stream = SchemaMigrator.class.getResourceAsStream(resource)) {
            if (stream == null) {
                throw new PersistenceException("Missing schema migration " + resource);
            }
            return new String(stream.readAllBytes(), StandardCharsets.UTF_8);
        } catch (IOException exception) {
            throw new PersistenceException(
                "Could not read schema migration " + resource,
                exception
            );
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

    private record Migration(int version, String description, String resource) {
    }
}
