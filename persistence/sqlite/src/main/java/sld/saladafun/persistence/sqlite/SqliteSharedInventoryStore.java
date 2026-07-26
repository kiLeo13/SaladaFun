package sld.saladafun.persistence.sqlite;

import org.jooq.DSLContext;
import org.jooq.Record;
import sld.saladafun.shared.inventory.model.InitialInventoryMode;
import sld.saladafun.shared.inventory.model.InventorySnapshot;
import sld.saladafun.shared.inventory.model.ItemFingerprint;
import sld.saladafun.shared.inventory.model.ItemKey;
import sld.saladafun.shared.inventory.model.ItemStackSnapshot;
import sld.saladafun.shared.inventory.model.PlayerInventoryBackup;
import sld.saladafun.shared.inventory.model.ReplicaState;
import sld.saladafun.shared.inventory.model.SessionId;
import sld.saladafun.shared.inventory.model.SessionLabel;
import sld.saladafun.shared.inventory.model.SessionStatus;
import sld.saladafun.shared.inventory.model.SharedInventorySession;
import sld.saladafun.shared.inventory.model.SlotKey;
import sld.saladafun.shared.inventory.repository.SharedInventoryStore;

import java.nio.file.Path;
import java.time.Instant;
import java.time.LocalDate;
import java.time.format.DateTimeFormatter;
import java.util.ArrayList;
import java.util.EnumMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Optional;
import java.util.UUID;

/**
 * jOOQ-backed SQLite implementation of the shared-inventory persistence port.
 */
public final class SqliteSharedInventoryStore implements SharedInventoryStore {
    private static final DateTimeFormatter LABEL_DATE = DateTimeFormatter.BASIC_ISO_DATE;
    private final SqliteDatabase database;

    public SqliteSharedInventoryStore(Path databaseFile) {
        database = new SqliteDatabase(databaseFile);
    }

    @Override
    public Optional<SharedInventorySession> loadActiveSession() {
        String id = value(
            database.context(),
            "SELECT active_session_id FROM shared_inventory_control WHERE singleton_id = 1",
            String.class
        );
        return id == null ? Optional.empty() : Optional.of(loadSession(database.context(), id));
    }

    @Override
    public SharedInventorySession createSession(
        LocalDate labelDate,
        InitialInventoryMode mode,
        UUID sourcePlayerId,
        InventorySnapshot initialInventory,
        Map<UUID, InventorySnapshot> onlineBackups
    ) {
        Objects.requireNonNull(labelDate, "labelDate");
        Objects.requireNonNull(mode, "mode");
        Objects.requireNonNull(initialInventory, "initialInventory");
        Objects.requireNonNull(onlineBackups, "onlineBackups");
        return database.context().transactionResult(configuration -> {
            DSLContext transaction = configuration.dsl();
            ensureNoActiveSession(transaction);
            String date = LABEL_DATE.format(labelDate);
            int sequence = nextDailySequence(transaction, date);
            SessionId id = SessionId.create();
            SessionLabel label = new SessionLabel("%s_%02d".formatted(date, sequence));
            Instant now = Instant.now();
            InventorySnapshot canonical = new InventorySnapshot(0, initialInventory.slots());

            transaction.execute(
                "INSERT INTO shared_inventory_session("
                    + "session_id, session_label, session_date, daily_sequence, status, "
                    + "initial_mode, source_player_uuid, revision, started_at, archived_at"
                    + ") VALUES (?, ?, ?, ?, 'ACTIVE', ?, ?, ?, ?, NULL)",
                id.value().toString(),
                label.value(),
                date,
                sequence,
                mode.name(),
                sourcePlayerId == null ? null : sourcePlayerId.toString(),
                canonical.revision(),
                now.toString()
            );
            transaction.execute(
                "UPDATE shared_inventory_control SET active_session_id = ?, updated_at = ? "
                    + "WHERE singleton_id = 1",
                id.value().toString(),
                now.toString()
            );
            insertSlots(transaction, "shared_inventory_slot", id, null, canonical);
            for (var backup : onlineBackups.entrySet()) {
                replaceBackup(transaction, id, backup.getKey(), backup.getValue(), now);
            }
            return new SharedInventorySession(
                id,
                label,
                SessionStatus.ACTIVE,
                mode,
                sourcePlayerId,
                now,
                null,
                canonical
            );
        });
    }

    @Override
    public SharedInventorySession resumeSession(
        SessionLabel label,
        Map<UUID, InventorySnapshot> onlineBackups
    ) {
        Objects.requireNonNull(label, "label");
        return database.context().transactionResult(configuration -> {
            DSLContext transaction = configuration.dsl();
            ensureNoActiveSession(transaction);
            Record record = transaction.fetchOne(
                "SELECT session_id FROM shared_inventory_session "
                    + "WHERE session_label = ? AND status = 'ARCHIVED'",
                label.value()
            );
            if (record == null) {
                throw new PersistenceException("Archived session not found: " + label.value());
            }
            String idValue = record.get("session_id", String.class);
            SessionId id = new SessionId(UUID.fromString(idValue));
            Instant now = Instant.now();
            transaction.execute(
                "UPDATE shared_inventory_session "
                    + "SET status = 'ACTIVE', initial_mode = 'RESUMED', archived_at = NULL "
                    + "WHERE session_id = ?",
                idValue
            );
            transaction.execute(
                "UPDATE shared_inventory_control SET active_session_id = ?, updated_at = ? "
                    + "WHERE singleton_id = 1",
                idValue,
                now.toString()
            );
            transaction.execute(
                "DELETE FROM player_inventory_replica WHERE session_id = ?",
                idValue
            );
            for (var backup : onlineBackups.entrySet()) {
                replaceBackup(transaction, id, backup.getKey(), backup.getValue(), now);
            }
            return loadSession(transaction, idValue);
        });
    }

    @Override
    public void saveCanonical(SessionId sessionId, InventorySnapshot inventory) {
        database.context().transaction(configuration -> {
            DSLContext transaction = configuration.dsl();
            int updated = transaction.execute(
                "UPDATE shared_inventory_session SET revision = ? WHERE session_id = ?",
                inventory.revision(),
                sessionId.value().toString()
            );
            if (updated != 1) {
                throw new PersistenceException("Unknown shared session " + sessionId.value());
            }
            transaction.execute(
                "DELETE FROM shared_inventory_slot WHERE session_id = ?",
                sessionId.value().toString()
            );
            insertSlots(transaction, "shared_inventory_slot", sessionId, null, inventory);
        });
    }

    @Override
    public void archive(SessionId sessionId) {
        database.context().transaction(configuration -> {
            DSLContext transaction = configuration.dsl();
            Instant now = Instant.now();
            transaction.execute(
                "UPDATE shared_inventory_session SET status = 'ARCHIVED', archived_at = ? "
                    + "WHERE session_id = ?",
                now.toString(),
                sessionId.value().toString()
            );
            transaction.execute(
                "UPDATE shared_inventory_control SET active_session_id = NULL, updated_at = ? "
                    + "WHERE singleton_id = 1 AND active_session_id = ?",
                now.toString(),
                sessionId.value().toString()
            );
        });
    }

    @Override
    public List<SharedInventorySession> listArchivedSessions() {
        return database.context().fetch(
            "SELECT session_id FROM shared_inventory_session "
                + "WHERE status = 'ARCHIVED' ORDER BY started_at DESC"
        ).map(record -> loadSession(
            database.context(), record.get("session_id", String.class)
        ));
    }

    @Override
    public boolean hasBackup(SessionId sessionId, UUID playerId) {
        return database.context().fetchOne(
            "SELECT 1 FROM player_inventory_backup WHERE session_id = ? AND player_uuid = ?",
            sessionId.value().toString(),
            playerId.toString()
        ) != null;
    }

    @Override
    public void saveBackupIfAbsent(
        SessionId sessionId,
        UUID playerId,
        InventorySnapshot personalInventory
    ) {
        database.context().transaction(configuration -> {
            DSLContext transaction = configuration.dsl();
            boolean exists = transaction.fetchOne(
                "SELECT 1 FROM player_inventory_backup "
                    + "WHERE session_id = ? AND player_uuid = ?",
                sessionId.value().toString(),
                playerId.toString()
            ) != null;
            if (!exists) {
                replaceBackup(
                    transaction, sessionId, playerId, personalInventory, Instant.now()
                );
            }
        });
    }

    @Override
    public Optional<PlayerInventoryBackup> findBackup(SessionId sessionId, UUID playerId) {
        Record record = database.context().fetchOne(
            "SELECT restore_status, captured_at, restored_at FROM player_inventory_backup "
                + "WHERE session_id = ? AND player_uuid = ?",
            sessionId.value().toString(),
            playerId.toString()
        );
        return record == null
            ? Optional.empty()
            : Optional.of(mapBackup(database.context(), sessionId, playerId, record));
    }

    @Override
    public Optional<PlayerInventoryBackup> findPendingRestore(UUID playerId) {
        Record record = database.context().fetchOne(
            "SELECT session_id, restore_status, captured_at, restored_at "
                + "FROM player_inventory_backup "
                + "WHERE player_uuid = ? AND restore_status = 'RESTORE_PENDING' "
                + "ORDER BY captured_at DESC LIMIT 1",
            playerId.toString()
        );
        if (record == null) {
            return Optional.empty();
        }
        SessionId sessionId = new SessionId(
            UUID.fromString(record.get("session_id", String.class))
        );
        return Optional.of(mapBackup(database.context(), sessionId, playerId, record));
    }

    @Override
    public void markRestorePending(SessionId sessionId) {
        database.context().execute(
            "UPDATE player_inventory_backup SET restore_status = 'RESTORE_PENDING', "
                + "restored_at = NULL WHERE session_id = ?",
            sessionId.value().toString()
        );
    }

    @Override
    public void markRestored(SessionId sessionId, UUID playerId) {
        database.context().execute(
            "UPDATE player_inventory_backup SET restore_status = 'RESTORED', restored_at = ? "
                + "WHERE session_id = ? AND player_uuid = ?",
            Instant.now().toString(),
            sessionId.value().toString(),
            playerId.toString()
        );
    }

    @Override
    public Optional<ReplicaState> findReplica(SessionId sessionId, UUID playerId) {
        Record record = database.context().fetchOne(
            "SELECT last_applied_revision, last_inventory_fingerprint, observed_at "
                + "FROM player_inventory_replica WHERE session_id = ? AND player_uuid = ?",
            sessionId.value().toString(),
            playerId.toString()
        );
        if (record == null) {
            return Optional.empty();
        }
        return Optional.of(new ReplicaState(
            sessionId,
            playerId,
            record.get("last_applied_revision", Long.class),
            record.get("last_inventory_fingerprint", String.class),
            Instant.parse(record.get("observed_at", String.class))
        ));
    }

    @Override
    public void saveReplica(ReplicaState replica) {
        database.context().execute(
            "INSERT INTO player_inventory_replica("
                + "session_id, player_uuid, last_applied_revision, "
                + "last_inventory_fingerprint, observed_at"
                + ") VALUES (?, ?, ?, ?, ?) "
                + "ON CONFLICT(session_id, player_uuid) DO UPDATE SET "
                + "last_applied_revision = excluded.last_applied_revision, "
                + "last_inventory_fingerprint = excluded.last_inventory_fingerprint, "
                + "observed_at = excluded.observed_at",
            replica.sessionId().value().toString(),
            replica.playerId().toString(),
            replica.appliedRevision(),
            replica.inventoryFingerprint(),
            replica.observedAt().toString()
        );
    }

    @Override
    public void close() {
        database.close();
    }

    private SharedInventorySession loadSession(DSLContext context, String sessionId) {
        Record record = context.fetchOne(
            "SELECT * FROM shared_inventory_session WHERE session_id = ?",
            sessionId
        );
        if (record == null) {
            throw new PersistenceException("Unknown shared session " + sessionId);
        }
        long revision = record.get("revision", Long.class);
        return new SharedInventorySession(
            new SessionId(UUID.fromString(sessionId)),
            new SessionLabel(record.get("session_label", String.class)),
            SessionStatus.valueOf(record.get("status", String.class)),
            InitialInventoryMode.valueOf(record.get("initial_mode", String.class)),
            nullableUuid(record.get("source_player_uuid", String.class)),
            Instant.parse(record.get("started_at", String.class)),
            nullableInstant(record.get("archived_at", String.class)),
            loadSlots(
                context,
                "shared_inventory_slot",
                "session_id = ?",
                revision,
                sessionId
            )
        );
    }

    private PlayerInventoryBackup mapBackup(
        DSLContext context,
        SessionId sessionId,
        UUID playerId,
        Record record
    ) {
        return new PlayerInventoryBackup(
            sessionId,
            playerId,
            loadSlots(
                context,
                "player_inventory_backup_slot",
                "session_id = ? AND player_uuid = ?",
                0,
                sessionId.value().toString(),
                playerId.toString()
            ),
            PlayerInventoryBackup.RestoreStatus.valueOf(
                record.get("restore_status", String.class)
            ),
            Instant.parse(record.get("captured_at", String.class)),
            nullableInstant(record.get("restored_at", String.class))
        );
    }

    private InventorySnapshot loadSlots(
        DSLContext context,
        String table,
        String predicate,
        long revision,
        Object... bindings
    ) {
        var slots = new EnumMap<SlotKey, ItemStackSnapshot>(SlotKey.class);
        context.fetch(
            "SELECT slot_key, item_key, stacking_fingerprint, item_amount, "
                + "maximum_stack_size, payload_format, item_payload FROM "
                + table + " WHERE " + predicate,
            bindings
        ).forEach(record -> slots.put(
            SlotKey.valueOf(record.get("slot_key", String.class)),
            new ItemStackSnapshot(
                new ItemKey(record.get("item_key", String.class)),
                new ItemFingerprint(record.get("stacking_fingerprint", String.class)),
                record.get("item_amount", Integer.class),
                record.get("maximum_stack_size", Integer.class),
                record.get("payload_format", String.class),
                record.get("item_payload", byte[].class)
            )
        ));
        return new InventorySnapshot(revision, slots);
    }

    private void replaceBackup(
        DSLContext context,
        SessionId sessionId,
        UUID playerId,
        InventorySnapshot inventory,
        Instant capturedAt
    ) {
        context.execute(
            "DELETE FROM player_inventory_backup WHERE session_id = ? AND player_uuid = ?",
            sessionId.value().toString(),
            playerId.toString()
        );
        context.execute(
            "INSERT INTO player_inventory_backup("
                + "session_id, player_uuid, restore_status, captured_at, restored_at"
                + ") VALUES (?, ?, 'CAPTURED', ?, NULL)",
            sessionId.value().toString(),
            playerId.toString(),
            capturedAt.toString()
        );
        insertSlots(
            context,
            "player_inventory_backup_slot",
            sessionId,
            playerId,
            inventory
        );
    }

    private void insertSlots(
        DSLContext context,
        String table,
        SessionId sessionId,
        UUID playerId,
        InventorySnapshot inventory
    ) {
        for (var slot : inventory.slots().entrySet()) {
            ItemStackSnapshot item = slot.getValue();
            if (playerId == null) {
                context.execute(
                    "INSERT INTO " + table + "("
                        + "session_id, slot_key, item_key, stacking_fingerprint, item_amount, "
                        + "maximum_stack_size, payload_format, item_payload, slot_revision"
                        + ") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
                    sessionId.value().toString(),
                    slot.getKey().name(),
                    item.itemKey().value(),
                    item.fingerprint().value(),
                    item.amount(),
                    item.maximumStackSize(),
                    item.payloadFormat(),
                    item.payload(),
                    inventory.revision()
                );
            } else {
                context.execute(
                    "INSERT INTO " + table + "("
                        + "session_id, player_uuid, slot_key, item_key, stacking_fingerprint, "
                        + "item_amount, maximum_stack_size, payload_format, item_payload"
                        + ") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
                    sessionId.value().toString(),
                    playerId.toString(),
                    slot.getKey().name(),
                    item.itemKey().value(),
                    item.fingerprint().value(),
                    item.amount(),
                    item.maximumStackSize(),
                    item.payloadFormat(),
                    item.payload()
                );
            }
        }
    }

    private int nextDailySequence(DSLContext context, String date) {
        Integer maximum = value(
            context,
            "SELECT MAX(daily_sequence) FROM shared_inventory_session WHERE session_date = ?",
            Integer.class,
            date
        );
        return maximum == null ? 1 : maximum + 1;
    }

    private void ensureNoActiveSession(DSLContext context) {
        String active = value(
            context,
            "SELECT active_session_id FROM shared_inventory_control WHERE singleton_id = 1",
            String.class
        );
        if (active != null) {
            throw new PersistenceException("A shared inventory session is already active");
        }
    }

    private static UUID nullableUuid(String value) {
        return value == null ? null : UUID.fromString(value);
    }

    private static Instant nullableInstant(String value) {
        return value == null ? null : Instant.parse(value);
    }

    private static <T> T value(
        DSLContext context,
        String sql,
        Class<T> type,
        Object... bindings
    ) {
        Record record = context.fetchOne(sql, bindings);
        return record == null ? null : record.get(0, type);
    }
}
