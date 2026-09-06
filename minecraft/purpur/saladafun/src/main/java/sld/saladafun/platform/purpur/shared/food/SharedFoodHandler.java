package sld.saladafun.platform.purpur.shared.food;

import com.destroystokyo.paper.event.player.PlayerPostRespawnEvent;
import com.destroystokyo.paper.event.server.ServerTickEndEvent;
import org.bukkit.entity.Player;
import org.bukkit.event.EventHandler;
import org.bukkit.event.EventPriority;
import org.bukkit.event.Listener;
import org.bukkit.event.entity.EntityExhaustionEvent;
import org.bukkit.event.entity.FoodLevelChangeEvent;
import org.bukkit.event.player.PlayerJoinEvent;
import org.bukkit.event.player.PlayerQuitEvent;
import sld.saladafun.shared.food.FoodContribution;
import sld.saladafun.shared.food.FoodState;
import sld.saladafun.shared.food.SharedFoodManager;

import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Objects;
import java.util.Set;
import java.util.UUID;

/** Purpur lifecycle and end-of-tick adapter for shared food. */
public final class SharedFoodHandler implements Listener {
    private final SharedFoodManager manager;
    private final PurpurFoodMapper mapper;
    private final PlayerFoodSynchronizer synchronizer;
    private final int safetyAuditIntervalTicks;
    private final List<FoodContribution> departingContributions = new ArrayList<>();
    private final Set<UUID> dirtyPlayers = new HashSet<>();
    private int ticksSinceAudit;

    public SharedFoodHandler(
        SharedFoodManager manager,
        PurpurFoodMapper mapper,
        PlayerFoodSynchronizer synchronizer,
        int safetyAuditIntervalTicks
    ) {
        this.manager = Objects.requireNonNull(manager, "manager");
        this.mapper = Objects.requireNonNull(mapper, "mapper");
        this.synchronizer = Objects.requireNonNull(synchronizer, "synchronizer");
        if (safetyAuditIntervalTicks < 1) {
            throw new IllegalArgumentException(
                "safetyAuditIntervalTicks must be positive"
            );
        }
        this.safetyAuditIntervalTicks = safetyAuditIntervalTicks;
    }

    @EventHandler(priority = EventPriority.LOWEST)
    public void onJoin(PlayerJoinEvent event) {
        reconcilePlayer(event.getPlayer());
    }

    public void reconcilePlayer(Player player) {
        manager.pendingRestore(player.getUniqueId()).ifPresent(backup -> {
            synchronizer.apply(player, backup.state());
            manager.markRestored(backup);
        });
        if (!manager.isEnabled()) {
            synchronizer.forget(player.getUniqueId());
            return;
        }
        FoodState personal = mapper.snapshot(player, 0);
        FoodState canonical = manager.join(player.getUniqueId(), personal);
        synchronizer.apply(player, canonical);
    }

    @EventHandler(priority = EventPriority.LOWEST)
    public void onQuit(PlayerQuitEvent event) {
        if (manager.isEnabled()) {
            synchronizer.observe(event.getPlayer()).ifPresent(departingContributions::add);
        }
        synchronizer.forget(event.getPlayer().getUniqueId());
    }

    @EventHandler(priority = EventPriority.LOWEST)
    public void onPostRespawn(PlayerPostRespawnEvent event) {
        reconcilePlayer(event.getPlayer());
    }

    @EventHandler(priority = EventPriority.LOWEST)
    public void onFoodLevelChange(FoodLevelChangeEvent event) {
        if (manager.isEnabled() && event.getEntity() instanceof Player player) {
            dirtyPlayers.add(player.getUniqueId());
        }
    }

    @EventHandler(priority = EventPriority.LOWEST)
    public void onExhaustion(EntityExhaustionEvent event) {
        if (manager.isEnabled() && event.getEntity() instanceof Player player) {
            dirtyPlayers.add(player.getUniqueId());
        }
    }

    @EventHandler(priority = EventPriority.LOW)
    public void onTickEnd(ServerTickEndEvent event) {
        if (!manager.isEnabled()) {
            departingContributions.clear();
            dirtyPlayers.clear();
            ticksSinceAudit = 0;
            return;
        }
        ticksSinceAudit++;
        if (ticksSinceAudit >= safetyAuditIntervalTicks) {
            synchronizer.onlinePlayerIds().forEach(dirtyPlayers::add);
            ticksSinceAudit = 0;
        }
        Set<UUID> observedPlayers = Set.copyOf(dirtyPlayers);
        dirtyPlayers.clear();
        if (observedPlayers.isEmpty() && departingContributions.isEmpty()) {
            return;
        }
        var contributions = new ArrayList<>(departingContributions);
        contributions.addAll(synchronizer.observePlayers(observedPlayers));
        departingContributions.clear();
        if (contributions.isEmpty()) {
            return;
        }
        long previousRevision = manager.current().orElseThrow().revision();
        FoodState canonical = manager.applyTick(List.copyOf(contributions));
        if (canonical.revision() != previousRevision) {
            synchronizer.applyToAll(canonical);
        } else if (!observedPlayers.isEmpty()) {
            synchronizer.applyToPlayers(observedPlayers, canonical);
        }
    }

    public void resetReplicas() {
        synchronizer.clear();
        departingContributions.clear();
        dirtyPlayers.clear();
        ticksSinceAudit = 0;
    }
}
