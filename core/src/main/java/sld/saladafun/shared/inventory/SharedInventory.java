package sld.saladafun.shared.inventory;

import sld.saladafun.shared.inventory.model.InventorySnapshot;
import sld.saladafun.shared.inventory.model.ItemFingerprint;
import sld.saladafun.shared.inventory.model.ItemStackSnapshot;
import sld.saladafun.shared.inventory.model.MutationResult;
import sld.saladafun.shared.inventory.model.MutationStatus;
import sld.saladafun.shared.inventory.model.OperationContext;
import sld.saladafun.shared.inventory.model.OperationId;
import sld.saladafun.shared.inventory.model.SlotKey;

import java.util.EnumMap;
import java.util.HashMap;
import java.util.HashSet;
import java.util.LinkedHashSet;
import java.util.Map;
import java.util.Objects;
import java.util.Set;

/**
 * Thread-safe authoritative aggregate for the shared inventory.
 *
 * <p>All public mutation methods are synchronized and perform only in-memory domain work.
 * Callers must persist returned snapshots and update platform inventories after releasing
 * this object's monitor.</p>
 */
public final class SharedInventory {
    private static final int COMPLETED_OPERATION_CACHE_SIZE = 8_192;
    private final EnumMap<SlotKey, ItemStackSnapshot> committedSlots;
    private final EnumMap<SlotKey, ItemStackSnapshot> effectiveSlots;
    private final Map<OperationId, Reservation> reservations = new HashMap<>();
    private final Set<OperationId> completedOperations = new LinkedHashSet<>();
    private final Set<SlotKey> reservedSlots = new HashSet<>();
    private long revision;

    public SharedInventory(InventorySnapshot initialState) {
        Objects.requireNonNull(initialState, "initialState");
        revision = initialState.revision();
        committedSlots = new EnumMap<>(SlotKey.class);
        committedSlots.putAll(initialState.slots());
        effectiveSlots = new EnumMap<>(committedSlots);
    }

    /**
     * Returns the last committed state. Pending reservations are deliberately not exposed.
     */
    public synchronized InventorySnapshot snapshot() {
        return new InventorySnapshot(revision, committedSlots);
    }

    /**
     * Applies an ordinary slot mutation only when the caller's old value is still canonical.
     */
    public synchronized MutationResult compareAndSetSlot(
        SlotKey slot,
        ItemStackSnapshot expected,
        ItemStackSnapshot replacement,
        OperationContext context
    ) {
        Objects.requireNonNull(slot, "slot");
        Objects.requireNonNull(context, "context");
        if (reservedSlots.contains(slot)) {
            return result(MutationStatus.SLOT_RESERVED, 0, 0);
        }
        if (!Objects.equals(effectiveSlots.get(slot), expected)) {
            return result(MutationStatus.STALE_STATE, 0, 0);
        }
        putOrRemove(committedSlots, slot, replacement);
        putOrRemove(effectiveSlots, slot, replacement);
        revision++;
        return result(MutationStatus.ACCEPTED, replacement == null ? 0 : replacement.amount(), 0);
    }

    /**
     * Immediately makes an item quantity unavailable while deferring the durable mutation
     * until the surrounding platform event has reached its final outcome.
     */
    public synchronized MutationResult reserveRemoval(
        OperationId operationId,
        SlotKey slot,
        ItemFingerprint expectedFingerprint,
        int requestedAmount,
        OperationContext context
    ) {
        validateReservationArguments(operationId, context, requestedAmount);
        Objects.requireNonNull(slot, "slot");
        Objects.requireNonNull(expectedFingerprint, "expectedFingerprint");
        MutationResult duplicate = duplicateResult(operationId, requestedAmount);
        if (duplicate != null) {
            return duplicate;
        }
        if (reservedSlots.contains(slot)) {
            return result(MutationStatus.SLOT_RESERVED, 0, requestedAmount);
        }
        ItemStackSnapshot existing = effectiveSlots.get(slot);
        if (existing == null
            || !existing.fingerprint().equals(expectedFingerprint)
            || existing.amount() < requestedAmount) {
            return result(MutationStatus.ITEM_UNAVAILABLE, 0, requestedAmount);
        }

        ItemStackSnapshot replacement = existing.amount() == requestedAmount
            ? null
            : existing.withAmount(existing.amount() - requestedAmount);
        reserve(
            operationId,
            Map.of(slot, existing),
            singletonNullable(slot, replacement),
            Set.of(slot),
            requestedAmount,
            0
        );
        return result(MutationStatus.ACCEPTED, requestedAmount, 0);
    }

    /**
     * Reserves as much of the requested quantity as storage and hotbar capacity permit.
     */
    public synchronized MutationResult reserveInsertion(
        OperationId operationId,
        ItemStackSnapshot item,
        int requestedAmount,
        OperationContext context
    ) {
        validateReservationArguments(operationId, context, requestedAmount);
        Objects.requireNonNull(item, "item");
        MutationResult duplicate = duplicateResult(operationId, requestedAmount);
        if (duplicate != null) {
            return duplicate;
        }

        var before = new EnumMap<SlotKey, ItemStackSnapshot>(SlotKey.class);
        var after = new EnumMap<SlotKey, ItemStackSnapshot>(SlotKey.class);
        var touched = new HashSet<SlotKey>();
        int remaining = requestedAmount;

        for (SlotKey slot : SlotKey.storageSlots()) {
            if (remaining == 0) {
                break;
            }
            ItemStackSnapshot existing = effectiveSlots.get(slot);
            if (existing == null || reservedSlots.contains(slot) || !existing.canStackWith(item)) {
                continue;
            }
            int capacity = existing.maximumStackSize() - existing.amount();
            if (capacity == 0) {
                continue;
            }
            int inserted = Math.min(capacity, remaining);
            before.put(slot, existing);
            after.put(slot, existing.withAmount(existing.amount() + inserted));
            touched.add(slot);
            remaining -= inserted;
        }

        for (SlotKey slot : SlotKey.storageSlots()) {
            if (remaining == 0) {
                break;
            }
            if (effectiveSlots.containsKey(slot) || reservedSlots.contains(slot)) {
                continue;
            }
            int inserted = Math.min(item.maximumStackSize(), remaining);
            after.put(slot, item.withAmount(inserted));
            touched.add(slot);
            remaining -= inserted;
        }

        int accepted = requestedAmount - remaining;
        if (accepted == 0) {
            return result(MutationStatus.INVENTORY_FULL, 0, requestedAmount);
        }
        reserve(operationId, before, after, touched, accepted, remaining);
        return result(MutationStatus.ACCEPTED, accepted, remaining);
    }

    /**
     * Reserves all occupied slots, used to serialize global death behavior.
     */
    public synchronized MutationResult reserveClear(
        OperationId operationId,
        OperationContext context
    ) {
        Objects.requireNonNull(operationId, "operationId");
        Objects.requireNonNull(context, "context");
        MutationResult duplicate = duplicateResult(operationId, 0);
        if (duplicate != null) {
            return duplicate;
        }
        if (!reservedSlots.isEmpty()) {
            return result(MutationStatus.SLOT_RESERVED, 0, 0);
        }
        var before = new EnumMap<SlotKey, ItemStackSnapshot>(committedSlots);
        int amount = before.values().stream().mapToInt(ItemStackSnapshot::amount).sum();
        reserve(operationId, before, Map.of(), before.keySet(), amount, 0);
        return result(MutationStatus.ACCEPTED, amount, 0);
    }

    /**
     * Commits a previously accepted reservation as one new canonical revision.
     */
    public synchronized MutationResult commit(OperationId operationId) {
        Objects.requireNonNull(operationId, "operationId");
        if (completedOperations.contains(operationId)) {
            return result(MutationStatus.ALREADY_COMPLETED, 0, 0);
        }
        Reservation reservation = reservations.remove(operationId);
        if (reservation == null) {
            return result(MutationStatus.UNKNOWN_OPERATION, 0, 0);
        }
        applyChanges(committedSlots, reservation.touchedSlots(), reservation.after());
        reservedSlots.removeAll(reservation.touchedSlots());
        rememberCompleted(operationId);
        revision++;
        return result(
            MutationStatus.ACCEPTED,
            reservation.acceptedAmount(),
            reservation.remainingAmount()
        );
    }

    /**
     * Releases a reservation without changing the canonical revision.
     */
    public synchronized MutationResult rollback(OperationId operationId) {
        Objects.requireNonNull(operationId, "operationId");
        if (completedOperations.contains(operationId)) {
            return result(MutationStatus.ALREADY_COMPLETED, 0, 0);
        }
        Reservation reservation = reservations.remove(operationId);
        if (reservation == null) {
            return result(MutationStatus.UNKNOWN_OPERATION, 0, 0);
        }
        applyChanges(effectiveSlots, reservation.touchedSlots(), reservation.before());
        reservedSlots.removeAll(reservation.touchedSlots());
        rememberCompleted(operationId);
        return result(
            MutationStatus.ACCEPTED,
            reservation.acceptedAmount(),
            reservation.remainingAmount()
        );
    }

    /**
     * Promotes a complete external snapshot as the next LWW revision.
     */
    public synchronized MutationResult promote(InventorySnapshot external) {
        Objects.requireNonNull(external, "external");
        if (!reservations.isEmpty()) {
            return result(MutationStatus.SLOT_RESERVED, 0, 0);
        }
        committedSlots.clear();
        committedSlots.putAll(external.slots());
        effectiveSlots.clear();
        effectiveSlots.putAll(external.slots());
        revision = Math.max(revision, external.revision()) + 1;
        return result(MutationStatus.ACCEPTED, 0, 0);
    }

    private void reserve(
        OperationId id,
        Map<SlotKey, ItemStackSnapshot> before,
        Map<SlotKey, ItemStackSnapshot> after,
        Set<SlotKey> touched,
        int accepted,
        int remaining
    ) {
        applyChanges(effectiveSlots, touched, after);
        reservedSlots.addAll(touched);
        reservations.put(
            id,
            new Reservation(
                Map.copyOf(before), Map.copyOf(after), Set.copyOf(touched), accepted, remaining
            )
        );
    }

    private MutationResult duplicateResult(OperationId id, int requested) {
        Reservation existing = reservations.get(id);
        if (existing != null) {
            return result(
                MutationStatus.ACCEPTED, existing.acceptedAmount(), existing.remainingAmount()
            );
        }
        if (completedOperations.contains(id)) {
            return result(MutationStatus.ALREADY_COMPLETED, 0, requested);
        }
        return null;
    }

    private void rememberCompleted(OperationId operationId) {
        completedOperations.add(operationId);
        if (completedOperations.size() <= COMPLETED_OPERATION_CACHE_SIZE) {
            return;
        }
        var oldest = completedOperations.iterator();
        oldest.next();
        oldest.remove();
    }

    private void validateReservationArguments(
        OperationId id,
        OperationContext context,
        int requestedAmount
    ) {
        Objects.requireNonNull(id, "operationId");
        Objects.requireNonNull(context, "context");
        if (requestedAmount < 1) {
            throw new IllegalArgumentException("requestedAmount must be positive");
        }
    }

    private MutationResult result(MutationStatus status, int accepted, int remaining) {
        return new MutationResult(status, snapshot(), accepted, remaining);
    }

    private static Map<SlotKey, ItemStackSnapshot> singletonNullable(
        SlotKey slot,
        ItemStackSnapshot value
    ) {
        if (value == null) {
            return Map.of();
        }
        return Map.of(slot, value);
    }

    private static void applyChanges(
        EnumMap<SlotKey, ItemStackSnapshot> target,
        Set<SlotKey> touched,
        Map<SlotKey, ItemStackSnapshot> values
    ) {
        for (SlotKey slot : touched) {
            putOrRemove(target, slot, values.get(slot));
        }
    }

    private static void putOrRemove(
        EnumMap<SlotKey, ItemStackSnapshot> target,
        SlotKey slot,
        ItemStackSnapshot value
    ) {
        if (value == null) {
            target.remove(slot);
        } else {
            target.put(slot, value);
        }
    }

    private record Reservation(
        Map<SlotKey, ItemStackSnapshot> before,
        Map<SlotKey, ItemStackSnapshot> after,
        Set<SlotKey> touchedSlots,
        int acceptedAmount,
        int remainingAmount
    ) {
    }
}
