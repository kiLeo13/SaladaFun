package sld.saladafun.persistence.sqlite;

import org.jooq.DSLContext;
import org.jooq.SQLDialect;
import org.jooq.impl.DSL;
import org.sqlite.SQLiteConfig;
import org.sqlite.SQLiteDataSource;

import java.nio.file.Path;
import java.sql.SQLException;
import java.util.Objects;

/**
 * Owns the SQLite data source, jOOQ context, and schema lifecycle.
 */
public final class SqliteDatabase implements AutoCloseable {
    private final SQLiteDataSource dataSource;
    private final DSLContext context;

    public SqliteDatabase(Path databaseFile) {
        Objects.requireNonNull(databaseFile, "databaseFile");
        SQLiteConfig config = new SQLiteConfig();
        config.enforceForeignKeys(true);
        config.setJournalMode(SQLiteConfig.JournalMode.WAL);
        config.setSynchronous(SQLiteConfig.SynchronousMode.FULL);
        config.setBusyTimeout(5_000);

        dataSource = new SQLiteDataSource(config);
        dataSource.setUrl("jdbc:sqlite:" + databaseFile.toAbsolutePath());
        context = DSL.using(dataSource, SQLDialect.SQLITE);
        new SchemaMigrator().migrate(context);
    }

    public DSLContext context() {
        return context;
    }

    @Override
    public void close() {
        try (var connection = dataSource.getConnection();
             var statement = connection.createStatement()) {
            statement.execute("PRAGMA wal_checkpoint(TRUNCATE)");
        } catch (SQLException exception) {
            throw new PersistenceException("Could not checkpoint SQLite during shutdown", exception);
        }
    }
}
