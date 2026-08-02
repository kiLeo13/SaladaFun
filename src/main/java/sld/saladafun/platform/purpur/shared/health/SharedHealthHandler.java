package sld.saladafun.platform.purpur.shared.health;

import com.destroystokyo.paper.event.player.PlayerPostRespawnEvent;
import com.destroystokyo.paper.event.server.ServerTickEndEvent;
import org.bukkit.entity.Player;
import org.bukkit.event.EventHandler;
import org.bukkit.event.EventPriority;
import org.bukkit.event.Listener;
import org.bukkit.event.entity.PlayerDeathEvent;
import org.bukkit.event.player.PlayerJoinEvent;
import org.bukkit.event.player.PlayerQuitEvent;
import sld.saladafun.shared.health.HealthContribution;
import sld.saladafun.shared.health.HealthPhase;
import sld.saladafun.shared.health.HealthState;
import sld.saladafun.shared.health.SharedHealthManager;

import java.util.ArrayList;
import java.util.List;
import java.util.Objects;
import java.util.UUID;

/** Purpur lifecycle and end-of-tick adapter for shared health. */
public final class SharedHealthHandler implements Listener {
    private final SharedHealthManager manager;
    private final PurpurHealthMapper mapper;
    private final PlayerHealthSynchronizer synchronizer;
    private final SharedDeathCoordinator deathCoordinator;
    private final List<HealthContribution> departingContributions = new ArrayList<>();
    private boolean lethalTick;
    private UUID primaryDeath;

    public SharedHealthHandler(
        SharedHealthManager manager,
        PurpurHealthMapper mapper,
        PlayerHealthSynchronizer synchronizer,
        SharedDeathCoordinator deathCoordinator
    ) {
        this.manager = Objects.requireNonNull(manager, "manager");
        this.mapper = Objects.requireNonNull(mapper, "mapper");
        this.synchronizer = Objects.requireNonNull(synchronizer, "synchronizer");
        this.deathCoordinator = Objects.requireNonNull(
            deathCoordinator, "deathCoordinator"
        );
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
        HealthState personal = mapper.snapshot(player, 0);
        HealthState canonical = manager.join(player.getUniqueId(), personal);
        synchronizer.apply(player, canonical);
        if (canonical.phase() == HealthPhase.DEAD && !player.isDead()) {
            player.setHealth(0.0);
        }
    }

    @EventHandler(priority = EventPriority.LOWEST)
    public void onQuit(PlayerQuitEvent event) {
        if (manager.isEnabled()) {
            synchronizer.observe(event.getPlayer()).ifPresent(departingContributions::add);
        }
        synchronizer.forget(event.getPlayer().getUniqueId());
    }

    @EventHandler(priority = EventPriority.LOWEST)
    public void onDeath(PlayerDeathEvent event) {
        if (!manager.isEnabled()) {
            return;
        }
        UUID playerId = event.getPlayer().getUniqueId();
        boolean generated = deathCoordinator.consumeGeneratedDeath(playerId);
        lethalTick = true;
        if (!generated && primaryDeath == null) {
            primaryDeath = playerId;
        }
    }

    @EventHandler(priority = EventPriority.LOWEST)
    public void onPostRespawn(PlayerPostRespawnEvent event) {
        manager.pendingRestore(event.getPlayer().getUniqueId()).ifPresent(backup -> {
            synchronizer.apply(event.getPlayer(), backup.state());
            manager.markRestored(backup);
        });
        if (!manager.isEnabled()) {
            return;
        }
        HealthState canonical = manager.current().orElseThrow();
        if (canonical.phase() == HealthPhase.DEAD) {
            canonical = manager.revive();
        }
        synchronizer.applyToAll(canonical);
    }

    @EventHandler(priority = EventPriority.LOW)
    public void onTickEnd(ServerTickEndEvent event) {
        if (!manager.isEnabled()) {
            departingContributions.clear();
            lethalTick = false;
            primaryDeath = null;
            return;
        }
        var contributions = new ArrayList<>(departingContributions);
        contributions.addAll(synchronizer.observeOnline());
        departingContributions.clear();
        HealthState canonical = manager.applyTick(
            List.copyOf(contributions), lethalTick
        );
        if (canonical.phase() == HealthPhase.DEAD) {
            UUID primary = primaryDeath;
            if (primary == null && !manager.current().isEmpty()) {
                primary = new UUID(0, 0);
            }
            deathCoordinator.killOtherPlayers(primary);
        } else {
            synchronizer.applyToAll(canonical);
        }
        lethalTick = false;
        primaryDeath = null;
    }

    public void resetReplicas() {
        synchronizer.clear();
        deathCoordinator.clear();
        departingContributions.clear();
        lethalTick = false;
        primaryDeath = null;
    }
}
