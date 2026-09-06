package sld.saladafun.persistence.sqlite;

import org.jooq.DSLContext;
import org.jooq.Record;
import sld.saladafun.shared.effects.EffectState;
import sld.saladafun.shared.effects.EffectsBackup;
import sld.saladafun.shared.effects.EffectsSession;
import sld.saladafun.shared.effects.EffectsState;
import sld.saladafun.shared.effects.SharedEffectsRepository;
import sld.saladafun.shared.model.InitialStateMode;
import sld.saladafun.shared.model.RestoreStatus;
import sld.saladafun.shared.model.SessionId;
import sld.saladafun.shared.model.SessionLabel;
import sld.saladafun.shared.model.SessionStatus;

import java.time.Clock;
import java.time.Instant;
import java.time.LocalDate;
import java.util.LinkedHashMap;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Optional;
import java.util.UUID;

/** SQLite/jOOQ implementation of the shared-effects persistence port. */
public final class SqliteSharedEffectsRepository implements SharedEffectsRepository {
    private static final String SESSION_COLUMNS = """
        s.session_id, s.session_label, s.session_date, s.daily_sequence,
        s.status, s.initial_mode, s.source_player_uuid, s.revision,
        s.started_at, s.archived_at
        """;

    private final DSLContext context;
    private final Clock clock;

    public SqliteSharedEffectsRepository(DSLContext context, Clock clock) {
        this.context = Objects.requireNonNull(context, "context");
        this.clock = Objects.requireNonNull(clock, "clock");
    }

    @Override
    public Optional<EffectsSession> loadActive() {
        Record row = context.fetchOne(
            "SELECT " + SESSION_COLUMNS + " FROM shared_effect_session s "
                + "JOIN shared_effect_control c ON c.active_session_id = s.session_id "
                + "WHERE c.singleton_id = 1"
        );
        return Optional.ofNullable(row).map(value -> effectsSession(context, value));
    }

    @Override
    public EffectsSession create(
        LocalDate labelDate,
        InitialStateMode mode,
        UUID sourcePlayerId,
        EffectsState initial,
        Map<UUID, EffectsState> backups
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
                "INSERT INTO shared_effect_session("
                    + "session_id, session_label, session_date, daily_sequence, "
                    + "status, initial_mode, source_player_uuid, revision, started_at"
                    + ") VALUES (?, ?, ?, ?, 'ACTIVE', ?, ?, ?, ?)",
                sessionId.value().toString(),
                label.value(),
                labelDate.toString(),
                sequence,
                mode.name(),
                sourcePlayerId == null ? null : sourcePlayerId.toString(),
                initial.revision(),
                now.toString()
            );
            replaceEffects(transaction, sessionId, initial.effects());
            transaction.execute(
                "UPDATE shared_effect_control SET active_session_id = ?, "
                    + "updated_at = ? WHERE singleton_id = 1",
                sessionId.value().toString(),
                now.toString()
            );
            backups.forEach((playerId, state) ->
                insertBackup(transaction, sessionId, playerId, state, now, false)
            );
            return new EffectsSession(
                sessionId, label, labelDate, sequence, SessionStatus.ACTIVE,
                mode, sourcePlayerId, initial, now, null
            );
        });
    }

    @Override
    public EffectsSession resume(
        SessionLabel label,
        Map<UUID, EffectsState> backups
    ) {
        Objects.requireNonNull(label, "label");
        Objects.requireNonNull(backups, "backups");
        Instant now = Instant.now(clock);
        return context.transactionResult(configuration -> {
            DSLContext transaction = configuration.dsl();
            requireDisabled(transaction);
            Record existing = transaction.fetchOne(
                "SELECT " + SESSION_COLUMNS + " FROM shared_effect_session s "
                    + "WHERE s.session_label = ? AND s.status = 'ARCHIVED'",
                label.value()
            );
            if (existing == null) {
                throw new IllegalArgumentException(
                    "Unknown archived effects session: " + label.value()
                );
            }
            EffectsSession archived = effectsSession(transaction, existing);
            transaction.execute(
                "UPDATE shared_effect_session SET status = 'ACTIVE', "
                    + "initial_mode = 'RESUMED', source_player_uuid = NULL, "
                    + "archived_at = NULL WHERE session_id = ?",
                archived.id().value().toString()
            );
            transaction.execute(
                "UPDATE shared_effect_control SET active_session_id = ?, "
                    + "updated_at = ? WHERE singleton_id = 1",
                archived.id().value().toString(),
                now.toString()
            );
            backups.forEach((playerId, state) ->
                insertBackup(transaction, archived.id(), playerId, state, now, true)
            );
            return new EffectsSession(
                archived.id(), archived.label(), archived.sessionDate(),
                archived.dailySequence(), SessionStatus.ACTIVE,
                InitialStateMode.RESUMED, null, archived.state(),
                archived.startedAt(), null
            );
        });
    }

    @Override
    public void saveCanonical(SessionId sessionId, EffectsState state) {
        context.transaction(configuration -> {
            DSLContext transaction = configuration.dsl();
            int updated = transaction.execute(
                "UPDATE shared_effect_session SET revision = ? "
                    + "WHERE session_id = ? AND status = 'ACTIVE'",
                state.revision(),
                sessionId.value().toString()
            );
            if (updated != 1) {
                throw new PersistenceException(
                    "Could not update active shared-effects state"
                );
            }
            replaceEffects(transaction, sessionId, state.effects());
        });
    }

    @Override
    public void archiveAndMarkRestores(SessionId sessionId) {
        Instant now = Instant.now(clock);
        context.transaction(configuration -> {
            DSLContext transaction = configuration.dsl();
            transaction.execute(
                "UPDATE shared_effect_session SET status = 'ARCHIVED', "
                    + "archived_at = ? WHERE session_id = ? AND status = 'ACTIVE'",
                now.toString(),
                sessionId.value().toString()
            );
            transaction.execute(
                "UPDATE player_effect_backup SET restore_status = 'RESTORE_PENDING' "
                    + "WHERE session_id = ? AND restore_status <> 'RESTORED'",
                sessionId.value().toString()
            );
            transaction.execute(
                "UPDATE shared_effect_control SET active_session_id = NULL, "
                    + "updated_at = ? WHERE singleton_id = 1 "
                    + "AND active_session_id = ?",
                now.toString(),
                sessionId.value().toString()
            );
        });
    }

    @Override
    public List<EffectsSession> listArchived() {
        return context.fetch(
            "SELECT " + SESSION_COLUMNS + " FROM shared_effect_session s "
                + "WHERE s.status = 'ARCHIVED' ORDER BY s.started_at DESC"
        ).map(row -> effectsSession(context, row));
    }

    @Override
    public void saveBackupIfAbsent(
        SessionId sessionId,
        UUID playerId,
        EffectsState state
    ) {
        context.transaction(configuration -> insertBackup(
            configuration.dsl(),
            sessionId,
            playerId,
            state,
            Instant.now(clock),
            false
        ));
    }

    @Override
    public Optional<EffectsBackup> findPendingRestore(UUID playerId) {
        Record row = context.fetchOne(
            "SELECT session_id, player_uuid, restore_status, captured_at, "
                + "restored_at FROM player_effect_backup WHERE player_uuid = ? "
                + "AND restore_status = 'RESTORE_PENDING' "
                + "ORDER BY captured_at DESC LIMIT 1",
            playerId.toString()
        );
        return Optional.ofNullable(row).map(value -> effectsBackup(context, value));
    }

    @Override
    public void markRestored(SessionId sessionId, UUID playerId) {
        context.execute(
            "UPDATE player_effect_backup SET restore_status = 'RESTORED', "
                + "restored_at = ? WHERE session_id = ? AND player_uuid = ?",
            Instant.now(clock).toString(),
            sessionId.value().toString(),
            playerId.toString()
        );
    }

    private void replaceEffects(
        DSLContext transaction,
        SessionId sessionId,
        Map<String, EffectState> effects
    ) {
        transaction.execute(
            "DELETE FROM shared_effect_value WHERE session_id = ?",
            sessionId.value().toString()
        );
        effects.values().forEach(effect -> {
            int layer = 0;
            EffectState current = effect;
            while (current != null) {
                transaction.execute(
                    "INSERT INTO shared_effect_value("
                        + "session_id, effect_type, layer_index, amplifier, "
                        + "duration_ticks, ambient, particles, icon) "
                        + "VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
                    sessionId.value().toString(),
                    current.typeKey(),
                    layer++,
                    current.amplifier(),
                    current.durationTicks(),
                    flag(current.ambient()),
                    flag(current.particles()),
                    flag(current.icon())
                );
                current = current.hiddenEffect();
            }
        });
    }

    private void insertBackup(
        DSLContext transaction,
        SessionId sessionId,
        UUID playerId,
        EffectsState state,
        Instant capturedAt,
        boolean replace
    ) {
        String conflict = replace
            ? "ON CONFLICT(session_id, player_uuid) DO UPDATE SET "
                + "restore_status = 'CAPTURED', captured_at = excluded.captured_at, "
                + "restored_at = NULL"
            : "ON CONFLICT(session_id, player_uuid) DO NOTHING";
        int inserted = transaction.execute(
            "INSERT INTO player_effect_backup("
                + "session_id, player_uuid, restore_status, captured_at"
                + ") VALUES (?, ?, 'CAPTURED', ?) " + conflict,
            sessionId.value().toString(),
            playerId.toString(),
            capturedAt.toString()
        );
        if (inserted == 0) {
            return;
        }
        transaction.execute(
            "DELETE FROM player_effect_backup_value "
                + "WHERE session_id = ? AND player_uuid = ?",
            sessionId.value().toString(),
            playerId.toString()
        );
        state.effects().values().forEach(effect -> {
            int layer = 0;
            EffectState current = effect;
            while (current != null) {
                transaction.execute(
                    "INSERT INTO player_effect_backup_value("
                        + "session_id, player_uuid, effect_type, layer_index, "
                        + "amplifier, duration_ticks, ambient, particles, icon) "
                        + "VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
                    sessionId.value().toString(),
                    playerId.toString(),
                    current.typeKey(),
                    layer++,
                    current.amplifier(),
                    current.durationTicks(),
                    flag(current.ambient()),
                    flag(current.particles()),
                    flag(current.icon())
                );
                current = current.hiddenEffect();
            }
        });
    }

    private EffectsSession effectsSession(DSLContext query, Record row) {
        SessionId id = new SessionId(UUID.fromString(row.get("session_id", String.class)));
        String source = row.get("source_player_uuid", String.class);
        String archived = row.get("archived_at", String.class);
        return new EffectsSession(
            id,
            new SessionLabel(row.get("session_label", String.class)),
            LocalDate.parse(row.get("session_date", String.class)),
            row.get("daily_sequence", Integer.class),
            SessionStatus.valueOf(row.get("status", String.class)),
            InitialStateMode.valueOf(row.get("initial_mode", String.class)),
            source == null ? null : UUID.fromString(source),
            new EffectsState(
                loadEffects(query, "shared_effect_value", id, null),
                row.get("revision", Long.class)
            ),
            Instant.parse(row.get("started_at", String.class)),
            archived == null ? null : Instant.parse(archived)
        );
    }

    private EffectsBackup effectsBackup(DSLContext query, Record row) {
        SessionId sessionId = new SessionId(
            UUID.fromString(row.get("session_id", String.class))
        );
        UUID playerId = UUID.fromString(row.get("player_uuid", String.class));
        String restored = row.get("restored_at", String.class);
        return new EffectsBackup(
            sessionId,
            playerId,
            new EffectsState(
                loadEffects(
                    query,
                    "player_effect_backup_value",
                    sessionId,
                    playerId
                ),
                0
            ),
            RestoreStatus.valueOf(row.get("restore_status", String.class)),
            Instant.parse(row.get("captured_at", String.class)),
            restored == null ? null : Instant.parse(restored)
        );
    }

    private Map<String, EffectState> loadEffects(
        DSLContext query,
        String table,
        SessionId sessionId,
        UUID playerId
    ) {
        String sql = "SELECT effect_type, layer_index, amplifier, duration_ticks, "
            + "ambient, particles, icon FROM " + table + " WHERE session_id = ?";
        Object[] parameters = {sessionId.value().toString()};
        if (playerId != null) {
            sql += " AND player_uuid = ?";
            parameters = new Object[]{sessionId.value().toString(), playerId.toString()};
        }
        sql += " ORDER BY effect_type, layer_index";
        Map<String, List<EffectLayer>> layers = new LinkedHashMap<>();
        for (Record effect : query.fetch(sql, parameters)) {
            String type = effect.get("effect_type", String.class);
            layers.computeIfAbsent(type, ignored -> new ArrayList<>()).add(
                new EffectLayer(
                    effect.get("layer_index", Integer.class),
                    effect.get("amplifier", Integer.class),
                    effect.get("duration_ticks", Integer.class),
                    effect.get("ambient", Integer.class) == 1,
                    effect.get("particles", Integer.class) == 1,
                    effect.get("icon", Integer.class) == 1
                )
            );
        }
        Map<String, EffectState> effects = new LinkedHashMap<>();
        layers.forEach((type, values) -> {
            for (int index = 0; index < values.size(); index++) {
                if (values.get(index).index() != index) {
                    throw new PersistenceException(
                        "Effect layer indexes must be contiguous for " + type
                    );
                }
            }
            EffectState hidden = null;
            for (int index = values.size() - 1; index >= 0; index--) {
                EffectLayer layer = values.get(index);
                hidden = new EffectState(
                    type,
                    layer.amplifier(),
                    layer.durationTicks(),
                    layer.ambient(),
                    layer.particles(),
                    layer.icon(),
                    hidden
                );
            }
            effects.put(type, hidden);
        });
        return Map.copyOf(effects);
    }

    private record EffectLayer(
        int index,
        int amplifier,
        int durationTicks,
        boolean ambient,
        boolean particles,
        boolean icon
    ) {
    }

    private void requireDisabled(DSLContext transaction) {
        String active = transaction.fetchOne(
            "SELECT active_session_id FROM shared_effect_control WHERE singleton_id = 1"
        ).get(0, String.class);
        if (active != null) {
            throw new IllegalStateException("Shared effects are already enabled");
        }
    }

    private int nextSequence(DSLContext transaction, LocalDate date) {
        Integer previous = transaction.fetchOne(
            "SELECT MAX(daily_sequence) FROM shared_effect_session "
                + "WHERE session_date = ?",
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

    private int flag(boolean value) {
        return value ? 1 : 0;
    }
}
