package sld.saladafun.persistence.sqlite;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import sld.saladafun.shared.inventory.SharedInventoryManager;
import sld.saladafun.shared.inventory.model.InitialInventoryMode;
import sld.saladafun.shared.inventory.model.InventorySnapshot;
import sld.saladafun.shared.inventory.model.ItemFingerprint;
import sld.saladafun.shared.inventory.model.ItemKey;
import sld.saladafun.shared.inventory.model.ItemStackSnapshot;
import sld.saladafun.shared.inventory.model.PlayerInventoryBackup;
import sld.saladafun.shared.inventory.model.ReplicaState;
import sld.saladafun.shared.inventory.model.SlotKey;

import java.nio.charset.StandardCharsets;
import java.nio.file.Path;
import java.time.Instant;
import java.time.LocalDate;
import java.time.Clock;
import java.time.ZoneOffset;
import java.util.Map;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.assertArrayEquals;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

class SqliteSharedInventoryStoreTest {
    @TempDir
    Path temporaryDirectory;

    @Test
    void createsHumanReadableLabelsAndPersistsOpaqueItems() {
        Path file = temporaryDirectory.resolve("inventory.db");
        UUID player = UUID.randomUUID();
        InventorySnapshot inventory = new InventorySnapshot(
            7, Map.of(SlotKey.OFF_HAND, item("totem_of_undying", 1, 1))
        );

        try (var store = new SqliteSharedInventoryStore(file)) {
            var first = store.createSession(
                LocalDate.of(2026, 7, 25),
                InitialInventoryMode.SOURCE_PLAYER,
                player,
                inventory,
                Map.of(player, inventory)
            );

            assertEquals("20260725_01", first.label().value());
            assertEquals(0, first.inventory().revision());
            assertArrayEquals(
                item("totem_of_undying", 1, 1).payload(),
                first.inventory().item(SlotKey.OFF_HAND).orElseThrow().payload()
            );
            store.markRestorePending(first.id());
            store.archive(first.id());

            var second = store.createSession(
                LocalDate.of(2026, 7, 25),
                InitialInventoryMode.EMPTY,
                null,
                InventorySnapshot.empty(0),
                Map.of()
            );
            assertEquals("20260725_02", second.label().value());
        }
    }

    @Test
    void roundTripsCanonicalBackupReplicaAndRestoreState() {
        Path file = temporaryDirectory.resolve("round-trip.db");
        UUID player = UUID.randomUUID();
        InventorySnapshot personal = new InventorySnapshot(
            0, Map.of(SlotKey.HOTBAR_0, item("diamond", 4, 64))
        );

        try (var store = new SqliteSharedInventoryStore(file)) {
            var session = store.createSession(
                LocalDate.of(2026, 7, 25),
                InitialInventoryMode.EMPTY,
                null,
                InventorySnapshot.empty(0),
                Map.of(player, personal)
            );
            InventorySnapshot canonical = new InventorySnapshot(
                9, Map.of(SlotKey.MAIN_0, item("stone", 64, 64))
            );
            store.saveCanonical(session.id(), canonical);
            store.saveReplica(new ReplicaState(
                session.id(), player, 9, canonical.fingerprint(), Instant.parse("2026-07-25T00:00:00Z")
            ));

            assertEquals(9, store.loadActiveSession().orElseThrow().inventory().revision());
            assertEquals(
                canonical.fingerprint(),
                store.findReplica(session.id(), player).orElseThrow().inventoryFingerprint()
            );

            store.markRestorePending(session.id());
            store.archive(session.id());
            PlayerInventoryBackup pending = store.findPendingRestore(player).orElseThrow();
            assertEquals(
                personal.fingerprint(),
                pending.inventory().fingerprint()
            );
            store.markRestored(session.id(), player);
            assertFalse(store.findPendingRestore(player).isPresent());
        }
    }

    @Test
    void existingDatabasePassesMigrationChecksumValidation() {
        Path file = temporaryDirectory.resolve("migration.db");
        try (var ignored = new SqliteSharedInventoryStore(file)) {
            assertTrue(file.toFile().exists());
        }
        try (var reopened = new SqliteSharedInventoryStore(file)) {
            assertTrue(reopened.loadActiveSession().isEmpty());
        }
    }

    @Test
    void joinReconciliationBacksUpNewPlayersAndPromotesProvenBukkitChanges() {
        UUID player = UUID.randomUUID();
        InventorySnapshot personal = new InventorySnapshot(
            0, Map.of(SlotKey.HOTBAR_0, item("diamond", 2, 64))
        );
        var store = new SqliteSharedInventoryStore(
            temporaryDirectory.resolve("manager.db")
        );
        try (var manager = new SharedInventoryManager(
            store,
            Clock.fixed(Instant.parse("2026-07-25T00:00:00Z"), ZoneOffset.UTC),
            ZoneOffset.UTC
        )) {
            manager.load();
            manager.enableEmpty(Map.of(player, personal));

            var firstJoin = manager.reconcileJoin(player, personal);
            assertEquals(
                SharedInventoryManager.JoinAction.APPLY_CANONICAL,
                firstJoin.action()
            );
            assertTrue(firstJoin.inventory().slots().isEmpty());

            manager.markReplicaApplied(player, firstJoin.inventory());
            var changedInBukkit = manager.reconcileJoin(player, personal);
            assertEquals(
                SharedInventoryManager.JoinAction.BUKKIT_PROMOTED,
                changedInBukkit.action()
            );
            assertEquals(1, changedInBukkit.inventory().revision());
            assertEquals(
                2,
                changedInBukkit.inventory().item(SlotKey.HOTBAR_0).orElseThrow().amount()
            );
        }
    }

    private ItemStackSnapshot item(String key, int amount, int maximum) {
        return new ItemStackSnapshot(
            new ItemKey("minecraft:" + key),
            new ItemFingerprint(key),
            amount,
            maximum,
            "test-nbt",
            key.getBytes(StandardCharsets.UTF_8)
        );
    }
}
