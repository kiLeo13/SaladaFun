package sld.saladafun.persistence.sqlite;

import org.jooq.DSLContext;
import org.jooq.Record;
import sld.saladafun.shared.health.HealthBackup;
import sld.saladafun.shared.health.HealthPhase;
import sld.saladafun.shared.health.HealthSession;
import sld.saladafun.shared.health.HealthState;
import sld.saladafun.shared.health.SharedHealthRepository;
import sld.saladafun.shared.model.InitialStateMode;
import sld.saladafun.shared.model.RestoreStatus;
import sld.saladafun.shared.model.SessionId;
import sld.saladafun.shared.model.SessionLabel;
import sld.saladafun.shared.model.SessionStatus;

import java.time.Clock;
import java.time.Instant;
import java.time.LocalDate;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Optional;
import java.util.UUID;

/** SQLite/jOOQ implementation of the shared-health persistence port. */
public final class SqliteSharedHealthRepository implements SharedHealthRepository {
    private static final String SESSION_COLUMNS = """
        s.session_id, s.session_label, s.session_date, s.daily_sequence,
        s.status, s.initial_mode, s.source_player_uuid, s.health,
        s.maximum_health, s.absorption, s.maximum_absorption, s.phase,
        s.revision, s.started_at, s.archived_at
        """;

    private final DSLContext context;
    private final Clock clock;

    public SqliteSharedHealthRepository(DSLContext context, Clock clock) {
        this.context = Objects.requireNonNull(context, "context");
        this.clock = Objects.requireNonNull(clock, "clock");
    }

    @Override
    public Optional<HealthSession> loadActive() {
        Record row = context.fetchOne(
            "SELECT " + SESSION_COLUMNS + " FROM shared_health_session s "
                + "JOIN shared_health_control c ON c.active_session_id = s.session_id "
                + "WHERE c.singleton_id = 1"
        );
        return Optional.ofNullable(row).map(this::healthSession);
    }

    @Override
    public HealthSession create(
        LocalDate labelDate,
        InitialStateMode mode,
        UUID sourcePlayerId,
        HealthState initial,
        Map<UUID, HealthState> backups
    ) {
        Objects.requireNonNull(labelDate, "labelDate");
        Objects.requireNonNull(mode, "mode");
        Objects.requireNonNull(initial, "initial");
        Objects.requireNonNull(backups, "backups");
        SessionId sessionId = SessionId.create();
        Instant now = Instant.now(clock);

        return context.transactionResult(configuration -> {
            DSLContext transaction = configuration.dsl();
            requireDisabled(transaction);
            int sequence = nextSequence(transaction, labelDate);
            SessionLabel label = label(labelDate, sequence);
            transaction.execute(
                "INSERT INTO shared_health_session("
                    + "session_id, session_label, session_date, daily_sequence, status, "
                    + "initial_mode, source_player_uuid, health, maximum_health, "
                    + "absorption, maximum_absorption, phase, revision, started_at"
                    + ") VALUES (?, ?, ?, ?, 'ACTIVE', ?, ?, ?, ?, ?, ?, ?, ?, ?)",
                sessionId.value().toString(),
                label.value(),
                labelDate.toString(),
                sequence,
                mode.name(),
                sourcePlayerId == null ? null : sourcePlayerId.toString(),
                initial.health(),
                initial.maximumHealth(),
                initial.absorption(),
                initial.maximumAbsorption(),
                initial.phase().name(),
                initial.revision(),
                now.toString()
            );
            transaction.execute(
                "UPDATE shared_health_control SET active_session_id = ?, updated_at = ? "
                    + "WHERE singleton_id = 1",
                sessionId.value().toString(),
                now.toString()
            );
            backups.forEach((playerId, state) ->
                insertBackup(transaction, sessionId, playerId, state, now, false)
            );
            return new HealthSession(
                sessionId,
                label,
                labelDate,
                sequence,
                SessionStatus.ACTIVE,
                mode,
                sourcePlayerId,
                initial,
                now,
                null
            );
        });
    }

    @Override
    public HealthSession resume(SessionLabel label, Map<UUID, HealthState> backups) {
        Objects.requireNonNull(label, "label");
        Objects.requireNonNull(backups, "backups");
        Instant now = Instant.now(clock);
        return context.transactionResult(configuration -> {
            DSLContext transaction = configuration.dsl();
            requireDisabled(transaction);
            Record existing = transaction.fetchOne(
                "SELECT " + SESSION_COLUMNS + " FROM shared_health_session s "
                    + "WHERE s.session_label = ? AND s.status = 'ARCHIVED'",
                label.value()
            );
            if (existing == null) {
                throw new IllegalArgumentException(
                    "Unknown archived health session: " + label.value()
                );
            }
            HealthSession archived = healthSession(existing);
            transaction.execute(
                "UPDATE shared_health_session SET status = 'ACTIVE', "
                    + "initial_mode = 'RESUMED', source_player_uuid = NULL, "
                    + "archived_at = NULL WHERE session_id = ?",
                archived.id().value().toString()
            );
            transaction.execute(
                "UPDATE shared_health_control SET active_session_id = ?, updated_at = ? "
                    + "WHERE singleton_id = 1",
                archived.id().value().toString(),
                now.toString()
            );
            backups.forEach((playerId, state) ->
                insertBackup(transaction, archived.id(), playerId, state, now, true)
            );
            return new HealthSession(
                archived.id(),
                archived.label(),
                archived.sessionDate(),
                archived.dailySequence(),
                SessionStatus.ACTIVE,
                InitialStateMode.RESUMED,
                null,
                archived.state(),
                archived.startedAt(),
                null
            );
        });
    }

    @Override
    public void saveCanonical(SessionId sessionId, HealthState state) {
        int updated = context.execute(
            "UPDATE shared_health_session SET health = ?, maximum_health = ?, "
                + "absorption = ?, maximum_absorption = ?, phase = ?, revision = ? "
                + "WHERE session_id = ? AND status = 'ACTIVE'",
            state.health(),
            state.maximumHealth(),
            state.absorption(),
            state.maximumAbsorption(),
            state.phase().name(),
            state.revision(),
            sessionId.value().toString()
        );
        if (updated != 1) {
            throw new PersistenceException("Could not update active shared-health state");
        }
    }

    @Override
    public void archiveAndMarkRestores(SessionId sessionId) {
        Instant now = Instant.now(clock);
        context.transaction(configuration -> {
            DSLContext transaction = configuration.dsl();
            transaction.execute(
                "UPDATE shared_health_session SET status = 'ARCHIVED', archived_at = ? "
                    + "WHERE session_id = ? AND status = 'ACTIVE'",
                now.toString(),
                sessionId.value().toString()
            );
            transaction.execute(
                "UPDATE player_health_backup SET restore_status = 'RESTORE_PENDING' "
                    + "WHERE session_id = ? AND restore_status <> 'RESTORED'",
                sessionId.value().toString()
            );
            transaction.execute(
                "UPDATE shared_health_control SET active_session_id = NULL, updated_at = ? "
                    + "WHERE singleton_id = 1 AND active_session_id = ?",
                now.toString(),
                sessionId.value().toString()
            );
        });
    }

    @Override
    public List<HealthSession> listArchived() {
        return context.fetch(
            "SELECT " + SESSION_COLUMNS + " FROM shared_health_session s "
                + "WHERE s.status = 'ARCHIVED' ORDER BY s.started_at DESC"
        ).map(this::healthSession);
    }

    @Override
    public void saveBackupIfAbsent(
        SessionId sessionId,
        UUID playerId,
        HealthState state
    ) {
        insertBackup(
            context,
            sessionId,
            playerId,
            state,
            Instant.now(clock),
            false
        );
    }

    @Override
    public Optional<HealthBackup> findPendingRestore(UUID playerId) {
        Record row = context.fetchOne(
            "SELECT session_id, player_uuid, health, maximum_health, absorption, "
                + "maximum_absorption, phase, restore_status, captured_at, restored_at "
                + "FROM player_health_backup WHERE player_uuid = ? "
                + "AND restore_status = 'RESTORE_PENDING' "
                + "ORDER BY captured_at DESC LIMIT 1",
            playerId.toString()
        );
        return Optional.ofNullable(row).map(this::healthBackup);
    }

    @Override
    public void markRestored(SessionId sessionId, UUID playerId) {
        context.execute(
            "UPDATE player_health_backup SET restore_status = 'RESTORED', restored_at = ? "
                + "WHERE session_id = ? AND player_uuid = ?",
            Instant.now(clock).toString(),
            sessionId.value().toString(),
            playerId.toString()
        );
    }

    private void requireDisabled(DSLContext transaction) {
        String active = transaction.fetchOne(
            "SELECT active_session_id FROM shared_health_control WHERE singleton_id = 1"
        ).get(0, String.class);
        if (active != null) {
            throw new IllegalStateException("Shared health is already enabled");
        }
    }

    private int nextSequence(DSLContext transaction, LocalDate date) {
        Integer previous = transaction.fetchOne(
            "SELECT MAX(daily_sequence) FROM shared_health_session WHERE session_date = ?",
            date.toString()
        ).get(0, Integer.class);
        return previous == null ? 1 : previous + 1;
    }

    private SessionLabel label(LocalDate date, int sequence) {
        return new SessionLabel("%s_%02d".formatted(
            date.toString().replace("-", ""),
            sequence
        ));
    }

    private void insertBackup(
        DSLContext transaction,
        SessionId sessionId,
        UUID playerId,
        HealthState state,
        Instant capturedAt,
        boolean replace
    ) {
        String conflict = replace
            ? "ON CONFLICT(session_id, player_uuid) DO UPDATE SET "
                + "health = excluded.health, maximum_health = excluded.maximum_health, "
                + "absorption = excluded.absorption, "
                + "maximum_absorption = excluded.maximum_absorption, "
                + "phase = excluded.phase, restore_status = 'CAPTURED', "
                + "captured_at = excluded.captured_at, restored_at = NULL"
            : "ON CONFLICT(session_id, player_uuid) DO NOTHING";
        transaction.execute(
            "INSERT INTO player_health_backup("
                + "session_id, player_uuid, health, maximum_health, absorption, "
                + "maximum_absorption, phase, restore_status, captured_at"
                + ") VALUES (?, ?, ?, ?, ?, ?, ?, 'CAPTURED', ?) " + conflict,
            sessionId.value().toString(),
            playerId.toString(),
            state.health(),
            state.maximumHealth(),
            state.absorption(),
            state.maximumAbsorption(),
            state.phase().name(),
            capturedAt.toString()
        );
    }

    private HealthSession healthSession(Record row) {
        String source = row.get("source_player_uuid", String.class);
        String archived = row.get("archived_at", String.class);
        return new HealthSession(
            new SessionId(UUID.fromString(row.get("session_id", String.class))),
            new SessionLabel(row.get("session_label", String.class)),
            LocalDate.parse(row.get("session_date", String.class)),
            row.get("daily_sequence", Integer.class),
            SessionStatus.valueOf(row.get("status", String.class)),
            InitialStateMode.valueOf(row.get("initial_mode", String.class)),
            source == null ? null : UUID.fromString(source),
            healthState(row),
            Instant.parse(row.get("started_at", String.class)),
            archived == null ? null : Instant.parse(archived)
        );
    }

    private HealthBackup healthBackup(Record row) {
        String restored = row.get("restored_at", String.class);
        return new HealthBackup(
            new SessionId(UUID.fromString(row.get("session_id", String.class))),
            UUID.fromString(row.get("player_uuid", String.class)),
            new HealthState(
                row.get("health", Double.class),
                row.get("maximum_health", Double.class),
                row.get("absorption", Double.class),
                row.get("maximum_absorption", Double.class),
                HealthPhase.valueOf(row.get("phase", String.class)),
                0
            ),
            RestoreStatus.valueOf(row.get("restore_status", String.class)),
            Instant.parse(row.get("captured_at", String.class)),
            restored == null ? null : Instant.parse(restored)
        );
    }

    private HealthState healthState(Record row) {
        return new HealthState(
            row.get("health", Double.class),
            row.get("maximum_health", Double.class),
            row.get("absorption", Double.class),
            row.get("maximum_absorption", Double.class),
            HealthPhase.valueOf(row.get("phase", String.class)),
            row.get("revision", Long.class)
        );
    }
}
