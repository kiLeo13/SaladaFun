package sld.saladafun.shared.inventory;

import org.junit.jupiter.api.Test;
import sld.saladafun.shared.inventory.model.InventorySnapshot;
import sld.saladafun.shared.inventory.model.ItemFingerprint;
import sld.saladafun.shared.inventory.model.ItemKey;
import sld.saladafun.shared.inventory.model.ItemStackSnapshot;
import sld.saladafun.shared.inventory.model.MutationStatus;
import sld.saladafun.shared.inventory.model.OperationContext;
import sld.saladafun.shared.inventory.model.OperationId;
import sld.saladafun.shared.inventory.model.SlotKey;

import java.nio.charset.StandardCharsets;
import java.util.Map;
import java.util.UUID;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.Executors;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertTrue;

class SharedInventoryTest {
    private static final OperationContext CONTEXT =
        new OperationContext(UUID.randomUUID(), "test");

    @Test
    void compareAndSetRejectsAStaleReplica() {
        ItemStackSnapshot stone = item("stone", 10, 64);
        SharedInventory inventory = inventory(Map.of(SlotKey.HOTBAR_0, stone));

        var accepted = inventory.compareAndSetSlot(
            SlotKey.HOTBAR_0, stone, stone.withAmount(9), CONTEXT
        );
        var rejected = inventory.compareAndSetSlot(
            SlotKey.HOTBAR_0, stone, stone.withAmount(8), CONTEXT
        );

        assertEquals(MutationStatus.ACCEPTED, accepted.status());
        assertEquals(MutationStatus.STALE_STATE, rejected.status());
        assertEquals(9, rejected.snapshot().item(SlotKey.HOTBAR_0).orElseThrow().amount());
        assertEquals(1, rejected.snapshot().revision());
    }

    @Test
    void reservationMakesTheSameItemsUnavailableUntilRollback() {
        ItemStackSnapshot stone = item("stone", 1, 64);
        SharedInventory inventory = inventory(Map.of(SlotKey.HOTBAR_0, stone));
        OperationId first = OperationId.create();

        assertTrue(inventory.reserveRemoval(
            first, SlotKey.HOTBAR_0, stone.fingerprint(), 1, CONTEXT
        ).accepted());
        assertEquals(
            MutationStatus.SLOT_RESERVED,
            inventory.reserveRemoval(
                OperationId.create(), SlotKey.HOTBAR_0, stone.fingerprint(), 1, CONTEXT
            ).status()
        );
        assertTrue(inventory.rollback(first).accepted());
        assertTrue(inventory.reserveRemoval(
            OperationId.create(), SlotKey.HOTBAR_0, stone.fingerprint(), 1, CONTEXT
        ).accepted());
    }

    @Test
    void simultaneousRemovalHasOnlyOneWinner() throws Exception {
        ItemStackSnapshot diamond = item("diamond", 1, 64);
        SharedInventory inventory = inventory(Map.of(SlotKey.HOTBAR_0, diamond));
        CountDownLatch start = new CountDownLatch(1);

        try (var executor = Executors.newFixedThreadPool(2)) {
            var first = executor.submit(() -> {
                start.await();
                return inventory.reserveRemoval(
                    OperationId.create(),
                    SlotKey.HOTBAR_0,
                    diamond.fingerprint(),
                    1,
                    CONTEXT
                ).accepted();
            });
            var second = executor.submit(() -> {
                start.await();
                return inventory.reserveRemoval(
                    OperationId.create(),
                    SlotKey.HOTBAR_0,
                    diamond.fingerprint(),
                    1,
                    CONTEXT
                ).accepted();
            });
            start.countDown();

            assertTrue(first.get() ^ second.get());
        }
    }

    @Test
    void insertionSupportsPartialPickupAndLeavesTheRemainder() {
        ItemStackSnapshot stone = item("stone", 63, 64);
        var slots = new java.util.EnumMap<SlotKey, ItemStackSnapshot>(SlotKey.class);
        for (SlotKey slot : SlotKey.storageSlots()) {
            slots.put(slot, item("dirt", 64, 64));
        }
        slots.put(SlotKey.HOTBAR_0, stone);
        SharedInventory inventory = inventory(slots);
        OperationId operation = OperationId.create();

        var reserved = inventory.reserveInsertion(
            operation, item("stone", 10, 64), 10, CONTEXT
        );
        assertTrue(reserved.accepted());
        assertEquals(1, reserved.acceptedAmount());
        assertEquals(9, reserved.remainingAmount());

        var committed = inventory.commit(operation);
        assertEquals(64, committed.snapshot().item(SlotKey.HOTBAR_0).orElseThrow().amount());
    }

    @Test
    void fullInventoryRejectsPickupWithoutChangingRevision() {
        var slots = new java.util.EnumMap<SlotKey, ItemStackSnapshot>(SlotKey.class);
        for (SlotKey slot : SlotKey.storageSlots()) {
            slots.put(slot, item("dirt", 64, 64));
        }
        SharedInventory inventory = inventory(slots);

        var result = inventory.reserveInsertion(
            OperationId.create(), item("stone", 1, 64), 1, CONTEXT
        );

        assertEquals(MutationStatus.INVENTORY_FULL, result.status());
        assertEquals(0, result.snapshot().revision());
    }

    @Test
    void committedClearRemovesEverySharedSlotExactlyOnce() {
        SharedInventory inventory = inventory(Map.of(
            SlotKey.HOTBAR_0, item("stone", 1, 64),
            SlotKey.ARMOR_HEAD, item("diamond_helmet", 1, 1)
        ));
        OperationId id = OperationId.create();

        assertTrue(inventory.reserveClear(id, CONTEXT).accepted());
        var committed = inventory.commit(id);

        assertTrue(committed.snapshot().slots().isEmpty());
        assertEquals(1, committed.snapshot().revision());
        assertEquals(MutationStatus.ALREADY_COMPLETED, inventory.commit(id).status());
    }

    @Test
    void payloadsAreDefensivelyCopied() {
        byte[] payload = "payload".getBytes(StandardCharsets.UTF_8);
        ItemStackSnapshot stack = new ItemStackSnapshot(
            new ItemKey("minecraft:stone"),
            new ItemFingerprint("stone"),
            1,
            64,
            "test",
            payload
        );
        payload[0] = 0;
        byte[] returned = stack.payload();
        returned[0] = 0;

        assertFalse(stack.payload()[0] == 0);
    }

    private static SharedInventory inventory(Map<SlotKey, ItemStackSnapshot> slots) {
        return new SharedInventory(new InventorySnapshot(0, slots));
    }

    private static ItemStackSnapshot item(String key, int amount, int maximum) {
        return new ItemStackSnapshot(
            new ItemKey("minecraft:" + key),
            new ItemFingerprint(key),
            amount,
            maximum,
            "test",
            key.getBytes(StandardCharsets.UTF_8)
        );
    }
}
