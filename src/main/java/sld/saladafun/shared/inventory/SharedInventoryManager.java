package sld.saladafun.shared.inventory;

import sld.saladafun.shared.inventory.model.InitialInventoryMode;
import sld.saladafun.shared.inventory.model.InventorySnapshot;
import sld.saladafun.shared.inventory.model.ItemFingerprint;
import sld.saladafun.shared.inventory.model.ItemStackSnapshot;
import sld.saladafun.shared.inventory.model.MutationResult;
import sld.saladafun.shared.inventory.model.MutationStatus;
import sld.saladafun.shared.inventory.model.OperationContext;
import sld.saladafun.shared.inventory.model.OperationId;
import sld.saladafun.shared.inventory.model.PlayerInventoryBackup;
import sld.saladafun.shared.inventory.model.ReplicaState;
import sld.saladafun.shared.inventory.model.SessionLabel;
import sld.saladafun.shared.inventory.model.SharedInventorySession;
import sld.saladafun.shared.inventory.model.SlotKey;
import sld.saladafun.shared.inventory.repository.SharedInventoryStore;

import java.time.Clock;
import java.time.Instant;
import java.time.LocalDate;
import java.time.ZoneId;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Optional;
import java.util.UUID;

/**
 * Application service and single owner of the active {@link SharedInventory} aggregate.
 */
public final class SharedInventoryManager implements AutoCloseable {
    private final SharedInventoryStore store;
    private final Clock clock;
    private final ZoneId labelZone;
    private SharedInventorySession activeSession;
    private SharedInventory activeInventory;

    public SharedInventoryManager(SharedInventoryStore store, Clock clock, ZoneId labelZone) {
        this.store = Objects.requireNonNull(store, "store");
        this.clock = Objects.requireNonNull(clock, "clock");
        this.labelZone = Objects.requireNonNull(labelZone, "labelZone");
    }

    /**
     * Loads a previously active session. Must be invoked before event registration.
     */
    public synchronized void load() {
        Optional<SharedInventorySession> loaded = store.loadActiveSession();
        activeSession = loaded.orElse(null);
        activeInventory = loaded.map(session -> new SharedInventory(session.inventory())).orElse(null);
    }

    public synchronized boolean isEnabled() {
        return activeInventory != null;
    }

    public synchronized Optional<InventorySnapshot> current() {
        return activeInventory == null ? Optional.empty() : Optional.of(activeInventory.snapshot());
    }

    public synchronized Optional<SharedInventorySession> activeSession() {
        return Optional.ofNullable(activeSession);
    }

    /**
     * Creates a fresh empty session. No player is selected implicitly.
     */
    public synchronized SharedInventorySession enableEmpty(
        Map<UUID, InventorySnapshot> onlineBackups
    ) {
        return enable(InitialInventoryMode.EMPTY, null, InventorySnapshot.empty(0), onlineBackups);
    }

    /**
     * Creates a fresh session using an explicitly selected online player's inventory.
     */
    public synchronized SharedInventorySession enableFrom(
        UUID sourcePlayerId,
        InventorySnapshot sourceInventory,
        Map<UUID, InventorySnapshot> onlineBackups
    ) {
        Objects.requireNonNull(sourcePlayerId, "sourcePlayerId");
        return enable(
            InitialInventoryMode.SOURCE_PLAYER,
            sourcePlayerId,
            sourceInventory,
            onlineBackups
        );
    }

    public synchronized SharedInventorySession resume(
        SessionLabel label,
        Map<UUID, InventorySnapshot> onlineBackups
    ) {
        requireDisabled();
        activeSession = store.resumeSession(label, Map.copyOf(onlineBackups));
        activeInventory = new SharedInventory(activeSession.inventory());
        return activeSession;
    }

    /**
     * Archives the canonical state and marks all personal backups for restoration.
     */
    public synchronized Optional<SharedInventorySession> disable() {
        if (activeInventory == null) {
            return Optional.empty();
        }
        InventorySnapshot finalState = activeInventory.snapshot();
        store.saveCanonical(activeSession.id(), finalState);
        store.markRestorePending(activeSession.id());
        store.archive(activeSession.id());
        SharedInventorySession disabled = activeSession;
        activeSession = null;
        activeInventory = null;
        return Optional.of(disabled);
    }

    /**
     * Reconciles the Bukkit inventory that became available on join.
     *
     * <p>A first-time participant is backed up and receives canonical state. A returning
     * participant promotes Bukkit only when its prior applied revision proves it is not
     * merely an expected stale offline replica.</p>
     */
    public synchronized JoinReconciliation reconcileJoin(
        UUID playerId,
        InventorySnapshot bukkitInventory
    ) {
        Objects.requireNonNull(playerId, "playerId");
        Objects.requireNonNull(bukkitInventory, "bukkitInventory");
        if (activeInventory == null) {
            return new JoinReconciliation(JoinAction.NO_SHARED_INVENTORY, bukkitInventory);
        }
        if (!store.hasBackup(activeSession.id(), playerId)) {
            store.saveBackupIfAbsent(activeSession.id(), playerId, bukkitInventory);
            return new JoinReconciliation(JoinAction.APPLY_CANONICAL, activeInventory.snapshot());
        }

        InventorySnapshot canonical = activeInventory.snapshot();
        Optional<ReplicaState> replica = store.findReplica(activeSession.id(), playerId);
        if (replica.isEmpty()) {
            // Without a last-applied marker there is no evidence that Bukkit is newer.
            // Applying canonical is the only choice that cannot resurrect an old replica.
            return new JoinReconciliation(JoinAction.APPLY_CANONICAL, canonical);
        }

        ReplicaState lastApplied = replica.orElseThrow();
        boolean contentChanged = !lastApplied.inventoryFingerprint()
            .equals(bukkitInventory.fingerprint());
        boolean bukkitAtLeastAsNew = lastApplied.appliedRevision() >= canonical.revision();
        if (contentChanged && bukkitAtLeastAsNew) {
            MutationResult promoted = activeInventory.promote(bukkitInventory);
            persistIfAccepted(promoted);
            return new JoinReconciliation(JoinAction.BUKKIT_PROMOTED, promoted.snapshot());
        }
        return new JoinReconciliation(JoinAction.APPLY_CANONICAL, canonical);
    }

    public synchronized Optional<PlayerInventoryBackup> pendingRestore(UUID playerId) {
        return store.findPendingRestore(playerId);
    }

    public synchronized void markRestored(PlayerInventoryBackup backup) {
        store.markRestored(backup.sessionId(), backup.playerId());
    }

    public synchronized void markReplicaApplied(UUID playerId, InventorySnapshot applied) {
        if (activeSession == null) {
            return;
        }
        store.saveReplica(
            new ReplicaState(
                activeSession.id(),
                playerId,
                applied.revision(),
                applied.fingerprint(),
                Instant.now(clock)
            )
        );
    }

    public synchronized MutationResult compareAndSetSlot(
        SlotKey slot,
        ItemStackSnapshot expected,
        ItemStackSnapshot replacement,
        OperationContext context
    ) {
        MutationResult result = requireInventory().compareAndSetSlot(
            slot, expected, replacement, context
        );
        persistIfAccepted(result);
        return result;
    }

    public synchronized MutationResult reserveRemoval(
        OperationId id,
        SlotKey slot,
        ItemFingerprint expected,
        int amount,
        OperationContext context
    ) {
        return requireInventory().reserveRemoval(id, slot, expected, amount, context);
    }

    public synchronized MutationResult reserveInsertion(
        OperationId id,
        ItemStackSnapshot item,
        int amount,
        OperationContext context
    ) {
        return requireInventory().reserveInsertion(id, item, amount, context);
    }

    public synchronized MutationResult reserveClear(
        OperationId id,
        OperationContext context
    ) {
        return requireInventory().reserveClear(id, context);
    }

    public synchronized MutationResult complete(OperationId id, boolean accepted) {
        MutationResult result = accepted
            ? requireInventory().commit(id)
            : requireInventory().rollback(id);
        if (accepted) {
            persistIfAccepted(result);
        }
        return result;
    }

    /**
     * Inserts retained death items after a committed global clear.
     */
    public synchronized InventorySnapshot insertRetainedItems(
        UUID actorId,
        List<ItemStackSnapshot> retained
    ) {
        for (ItemStackSnapshot item : retained) {
            OperationId id = OperationId.create();
            MutationResult reserved = activeInventory.reserveInsertion(
                id,
                item,
                item.amount(),
                new OperationContext(actorId, "death-retained-item")
            );
            if (reserved.accepted()) {
                MutationResult committed = activeInventory.commit(id);
                persistIfAccepted(committed);
            }
        }
        return activeInventory.snapshot();
    }

    public synchronized List<SharedInventorySession> archivedSessions() {
        return store.listArchivedSessions();
    }

    @Override
    public synchronized void close() {
        if (activeInventory != null) {
            store.saveCanonical(activeSession.id(), activeInventory.snapshot());
        }
        store.close();
    }

    private SharedInventorySession enable(
        InitialInventoryMode mode,
        UUID source,
        InventorySnapshot initial,
        Map<UUID, InventorySnapshot> backups
    ) {
        requireDisabled();
        LocalDate labelDate = LocalDate.now(clock.withZone(labelZone));
        activeSession = store.createSession(
            labelDate, mode, source, initial, Map.copyOf(backups)
        );
        activeInventory = new SharedInventory(activeSession.inventory());
        return activeSession;
    }

    private void persistIfAccepted(MutationResult result) {
        if (result.status() == MutationStatus.ACCEPTED) {
            store.saveCanonical(activeSession.id(), result.snapshot());
        }
    }

    private SharedInventory requireInventory() {
        if (activeInventory == null) {
            throw new IllegalStateException("Shared inventory is disabled");
        }
        return activeInventory;
    }

    private void requireDisabled() {
        if (activeInventory != null) {
            throw new IllegalStateException("Shared inventory is already enabled");
        }
    }

    public enum JoinAction {
        NO_SHARED_INVENTORY,
        APPLY_CANONICAL,
        BUKKIT_PROMOTED
    }

    public record JoinReconciliation(JoinAction action, InventorySnapshot inventory) {
        public JoinReconciliation {
            Objects.requireNonNull(action, "action");
            Objects.requireNonNull(inventory, "inventory");
        }
    }
}
