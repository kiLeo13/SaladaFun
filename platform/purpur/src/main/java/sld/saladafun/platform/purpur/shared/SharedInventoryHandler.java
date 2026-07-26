package sld.saladafun.platform.purpur.shared;

import com.destroystokyo.paper.event.player.PlayerPostRespawnEvent;
import io.papermc.paper.event.player.PlayerInventorySlotChangeEvent;
import org.bukkit.Bukkit;
import org.bukkit.entity.Player;
import org.bukkit.event.EventHandler;
import org.bukkit.event.EventPriority;
import org.bukkit.event.Listener;
import org.bukkit.event.entity.PlayerDeathEvent;
import org.bukkit.event.player.PlayerAttemptPickupItemEvent;
import org.bukkit.event.player.PlayerDropItemEvent;
import org.bukkit.event.player.PlayerJoinEvent;
import org.bukkit.inventory.ItemStack;
import org.bukkit.plugin.java.JavaPlugin;
import sld.saladafun.platform.purpur.config.DeathBehavior;
import sld.saladafun.platform.purpur.config.PluginSettings;
import sld.saladafun.shared.inventory.SharedInventoryManager;
import sld.saladafun.shared.inventory.model.InventorySnapshot;
import sld.saladafun.shared.inventory.model.ItemStackSnapshot;
import sld.saladafun.shared.inventory.model.MutationResult;
import sld.saladafun.shared.inventory.model.OperationContext;
import sld.saladafun.shared.inventory.model.OperationId;
import sld.saladafun.shared.inventory.model.SlotKey;

import java.util.IdentityHashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Optional;
import java.util.UUID;

/**
 * Purpur event facade for the shared-inventory application.
 *
 * <p>LOWEST handlers validate or reserve. MONITOR handlers are read-only with respect to
 * Bukkit events and only commit or roll back the earlier domain reservation.</p>
 */
public final class SharedInventoryHandler implements Listener {
    private final JavaPlugin plugin;
    private final SharedInventoryManager manager;
    private final PurpurInventoryMapper mapper;
    private final PlayerInventorySynchronizer synchronizer;
    private final PluginSettings settings;
    private final Map<PlayerDropItemEvent, OperationId> dropOperations = new IdentityHashMap<>();
    private final Map<PlayerAttemptPickupItemEvent, OperationId> pickupOperations =
        new IdentityHashMap<>();
    private final Map<PlayerDeathEvent, OperationId> deathOperations = new IdentityHashMap<>();
    private boolean synchronizationScheduled;

    public SharedInventoryHandler(
        JavaPlugin plugin,
        SharedInventoryManager manager,
        PurpurInventoryMapper mapper,
        PlayerInventorySynchronizer synchronizer,
        PluginSettings settings
    ) {
        this.plugin = Objects.requireNonNull(plugin, "plugin");
        this.manager = Objects.requireNonNull(manager, "manager");
        this.mapper = Objects.requireNonNull(mapper, "mapper");
        this.synchronizer = Objects.requireNonNull(synchronizer, "synchronizer");
        this.settings = Objects.requireNonNull(settings, "settings");
    }

    @EventHandler(priority = EventPriority.LOWEST)
    public void onJoin(PlayerJoinEvent event) {
        reconcilePlayer(event.getPlayer());
    }

    /**
     * Reconciles an inventory that is currently loaded by Bukkit.
     */
    public void reconcilePlayer(Player player) {
        manager.pendingRestore(player.getUniqueId()).ifPresent(backup -> {
            synchronizer.apply(player, backup.inventory());
            manager.markRestored(backup);
        });
        if (!manager.isEnabled()) {
            return;
        }

        InventorySnapshot loaded = mapper.snapshot(player, 0);
        SharedInventoryManager.JoinReconciliation result = manager.reconcileJoin(
            player.getUniqueId(), loaded
        );
        synchronizer.applyToAll(result.inventory());
    }

    @EventHandler(priority = EventPriority.LOWEST)
    public void onSlotChange(PlayerInventorySlotChangeEvent event) {
        Player player = event.getPlayer();
        if (!manager.isEnabled() || synchronizer.isSynchronizing(player)) {
            return;
        }
        Optional<SlotKey> mappedSlot = mapper.fromBukkitSlot(event.getSlot());
        if (mappedSlot.isEmpty()) {
            return;
        }
        MutationResult result = manager.compareAndSetSlot(
            mappedSlot.orElseThrow(),
            mapper.nullableSnapshot(event.getOldItemStack()),
            mapper.nullableSnapshot(event.getNewItemStack()),
            context(player, "slot-change")
        );
        if (result.accepted()) {
            scheduleSynchronization();
        } else {
            synchronizer.apply(player, result.snapshot());
        }
    }

    @EventHandler(priority = EventPriority.LOWEST, ignoreCancelled = true)
    public void reserveDrop(PlayerDropItemEvent event) {
        Player player = event.getPlayer();
        if (!manager.isEnabled() || synchronizer.isSynchronizing(player)) {
            return;
        }
        ItemStackSnapshot dropped = mapper.snapshot(event.getItemDrop().getItemStack());
        InventorySnapshot canonical = manager.current().orElseThrow();
        InventorySnapshot bukkit = mapper.snapshot(player, canonical.revision());
        Optional<SlotKey> candidate = removalSlot(
            player, canonical, bukkit, dropped
        );
        if (candidate.isEmpty()) {
            event.setCancelled(true);
            synchronizer.apply(player, canonical);
            return;
        }

        ItemStackSnapshot canonicalItem = canonical.item(candidate.orElseThrow()).orElseThrow();
        int currentAmount = bukkit.item(candidate.orElseThrow())
            .filter(item -> item.fingerprint().equals(dropped.fingerprint()))
            .map(ItemStackSnapshot::amount)
            .orElse(0);
        if (currentAmount == canonicalItem.amount()) {
            // The slot-change event already committed this drop.
            return;
        }
        if (canonicalItem.amount() - currentAmount != dropped.amount()) {
            event.setCancelled(true);
            synchronizer.apply(player, canonical);
            return;
        }

        OperationId id = OperationId.create();
        MutationResult reserved = manager.reserveRemoval(
            id,
            candidate.orElseThrow(),
            dropped.fingerprint(),
            dropped.amount(),
            context(player, "drop")
        );
        if (!reserved.accepted()) {
            event.setCancelled(true);
            synchronizer.apply(player, reserved.snapshot());
            return;
        }
        dropOperations.put(event, id);
    }

    @EventHandler(priority = EventPriority.MONITOR)
    public void completeDrop(PlayerDropItemEvent event) {
        OperationId id = dropOperations.remove(event);
        if (id == null) {
            return;
        }
        manager.complete(id, !event.isCancelled());
        scheduleSynchronization();
    }

    @EventHandler(priority = EventPriority.LOWEST, ignoreCancelled = true)
    public void reservePickup(PlayerAttemptPickupItemEvent event) {
        Player player = event.getPlayer();
        if (!manager.isEnabled() || synchronizer.isSynchronizing(player)) {
            return;
        }
        ItemStackSnapshot groundItem = mapper.snapshot(event.getItem().getItemStack());
        int vanillaAccepted = groundItem.amount() - event.getRemaining();
        if (vanillaAccepted <= 0) {
            event.setCancelled(true);
            return;
        }

        OperationId id = OperationId.create();
        MutationResult reserved = manager.reserveInsertion(
            id,
            groundItem,
            groundItem.amount(),
            context(player, "pickup")
        );
        if (!reserved.accepted() || reserved.acceptedAmount() != vanillaAccepted) {
            if (reserved.accepted()) {
                manager.complete(id, false);
            }
            event.setCancelled(true);
            return;
        }
        pickupOperations.put(event, id);
    }

    @EventHandler(priority = EventPriority.MONITOR)
    public void completePickup(PlayerAttemptPickupItemEvent event) {
        OperationId id = pickupOperations.remove(event);
        if (id == null) {
            return;
        }
        manager.complete(id, !event.isCancelled());
        scheduleSynchronization();
    }

    @EventHandler(priority = EventPriority.LOWEST)
    public void reserveDeath(PlayerDeathEvent event) {
        if (!manager.isEnabled()) {
            return;
        }
        DeathBehavior behavior = settings.deathBehavior();
        switch (behavior) {
            case FOLLOW_GAMERULE -> {
                // Observe the final keep-inventory value without overriding other plugins.
            }
            case DROPS_ON_DEATH -> event.setKeepInventory(false);
            case FADES_ON_DEATH -> {
                event.setKeepInventory(false);
                event.getDrops().clear();
            }
            case KEEPS_ON_DEATH -> {
                event.setKeepInventory(true);
                event.getDrops().clear();
            }
        }

        if (!event.getKeepInventory()) {
            OperationId id = OperationId.create();
            MutationResult reserved = manager.reserveClear(
                id, context(event.getPlayer(), "death")
            );
            if (reserved.accepted()) {
                deathOperations.put(event, id);
            }
        }
    }

    @EventHandler(priority = EventPriority.MONITOR)
    public void completeDeath(PlayerDeathEvent event) {
        if (!manager.isEnabled()) {
            return;
        }
        OperationId id = deathOperations.remove(event);
        if (event.getKeepInventory()) {
            if (id != null) {
                manager.complete(id, false);
            }
            scheduleSynchronization();
            return;
        }

        if (id == null) {
            id = OperationId.create();
            MutationResult reserved = manager.reserveClear(
                id, context(event.getPlayer(), "death-final")
            );
            if (!reserved.accepted()) {
                scheduleSynchronization();
                return;
            }
        }
        manager.complete(id, true);
        if (settings.respectItemsToKeep() && !event.getItemsToKeep().isEmpty()) {
            manager.insertRetainedItems(
                event.getPlayer().getUniqueId(),
                mapper.snapshots(event.getItemsToKeep())
            );
        }
        scheduleSynchronization();
    }

    @EventHandler(priority = EventPriority.LOWEST)
    public void onPostRespawn(PlayerPostRespawnEvent event) {
        manager.current().ifPresent(snapshot ->
            Bukkit.getScheduler().runTask(plugin, () ->
                synchronizer.apply(event.getPlayer(), snapshot)
            )
        );
    }

    private Optional<SlotKey> removalSlot(
        Player player,
        InventorySnapshot canonical,
        InventorySnapshot bukkit,
        ItemStackSnapshot dropped
    ) {
        SlotKey held = SlotKey.valueOf("HOTBAR_" + player.getInventory().getHeldItemSlot());
        if (matchesRemoval(held, canonical, bukkit, dropped)) {
            return Optional.of(held);
        }
        return canonical.slots().keySet().stream()
            .filter(slot -> matchesRemoval(slot, canonical, bukkit, dropped))
            .findFirst();
    }

    private boolean matchesRemoval(
        SlotKey slot,
        InventorySnapshot canonical,
        InventorySnapshot bukkit,
        ItemStackSnapshot dropped
    ) {
        ItemStackSnapshot canonicalItem = canonical.item(slot).orElse(null);
        if (canonicalItem == null
            || !canonicalItem.fingerprint().equals(dropped.fingerprint())) {
            return false;
        }
        int current = bukkit.item(slot)
            .filter(item -> item.fingerprint().equals(dropped.fingerprint()))
            .map(ItemStackSnapshot::amount)
            .orElse(0);
        return current == canonicalItem.amount()
            || canonicalItem.amount() - current == dropped.amount();
    }

    private OperationContext context(Player player, String type) {
        return new OperationContext(player.getUniqueId(), type);
    }

    private void scheduleSynchronization() {
        if (synchronizationScheduled) {
            return;
        }
        synchronizationScheduled = true;
        Bukkit.getScheduler().runTask(plugin, () -> {
            synchronizationScheduled = false;
            manager.current().ifPresent(synchronizer::applyToAll);
        });
    }
}
