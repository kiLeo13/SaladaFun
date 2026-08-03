package sld.saladafun.platform.purpur.shared.health;

import com.destroystokyo.paper.event.player.PlayerPostRespawnEvent;
import com.destroystokyo.paper.event.server.ServerTickEndEvent;
import io.papermc.paper.event.entity.EntityEquipmentChangedEvent;
import org.bukkit.damage.DamageSource;
import org.bukkit.entity.Player;
import org.bukkit.event.EventHandler;
import org.bukkit.event.EventPriority;
import org.bukkit.event.Listener;
import org.bukkit.event.entity.EntityDamageEvent;
import org.bukkit.event.entity.EntityPotionEffectEvent;
import org.bukkit.event.entity.EntityRegainHealthEvent;
import org.bukkit.event.entity.EntityResurrectEvent;
import org.bukkit.event.entity.PlayerDeathEvent;
import org.bukkit.event.player.PlayerJoinEvent;
import org.bukkit.event.player.PlayerQuitEvent;
import sld.saladafun.shared.health.HealthContribution;
import sld.saladafun.shared.health.HealthPhase;
import sld.saladafun.shared.health.HealthState;
import sld.saladafun.shared.health.SharedHealthManager;

import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Objects;
import java.util.Set;
import java.util.UUID;

/** Purpur lifecycle and end-of-tick adapter for shared health. */
public final class SharedHealthHandler implements Listener {
    private final SharedHealthManager manager;
    private final PurpurHealthMapper mapper;
    private final PlayerHealthSynchronizer synchronizer;
    private final SharedDeathCoordinator deathCoordinator;
    private final int safetyAuditIntervalTicks;
    private final List<HealthContribution> departingContributions = new ArrayList<>();
    private final List<PlayerDeathEvent> primaryDeaths = new ArrayList<>();
    private final List<EntityDamageEvent> playerDamageEvents = new ArrayList<>();
    private UUID deathWavePrimary;
    private DamageSource deathWaveSource;
    private final Set<UUID> dirtyPlayers = new HashSet<>();
    private int ticksSinceAudit;

    public SharedHealthHandler(
        SharedHealthManager manager,
        PurpurHealthMapper mapper,
        PlayerHealthSynchronizer synchronizer,
        SharedDeathCoordinator deathCoordinator,
        int safetyAuditIntervalTicks
    ) {
        this.manager = Objects.requireNonNull(manager, "manager");
        this.mapper = Objects.requireNonNull(mapper, "mapper");
        this.synchronizer = Objects.requireNonNull(synchronizer, "synchronizer");
        this.deathCoordinator = Objects.requireNonNull(
            deathCoordinator, "deathCoordinator"
        );
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
            synchronizer.restore(player, backup.state());
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
    public void onDamage(EntityDamageEvent event) {
        if (!manager.isEnabled() || !(event.getEntity() instanceof Player player)) {
            return;
        }
        dirtyPlayers.add(player.getUniqueId());
        if (manager.current()
            .map(state -> state.phase() == HealthPhase.DEAD)
            .orElse(false)) {
            return;
        }
        playerDamageEvents.add(event);
    }

    @EventHandler(priority = EventPriority.LOWEST)
    public void onRegainHealth(EntityRegainHealthEvent event) {
        markDirty(event.getEntity());
    }

    @EventHandler(priority = EventPriority.LOWEST)
    public void onPotionEffect(EntityPotionEffectEvent event) {
        markDirty(event.getEntity());
    }

    @EventHandler(priority = EventPriority.LOWEST)
    public void onEquipmentChanged(EntityEquipmentChangedEvent event) {
        markDirty(event.getEntity());
    }

    @EventHandler(priority = EventPriority.LOWEST)
    public void onResurrect(EntityResurrectEvent event) {
        markDirty(event.getEntity());
    }

    @EventHandler(priority = EventPriority.LOWEST)
    public void onDeath(PlayerDeathEvent event) {
        if (!manager.isEnabled()) {
            return;
        }
        UUID playerId = event.getPlayer().getUniqueId();
        dirtyPlayers.add(playerId);
        boolean generated = deathCoordinator.consumeGeneratedDeath(playerId);
        if (!generated) {
            primaryDeaths.add(event);
        }
    }

    @EventHandler(priority = EventPriority.LOWEST)
    public void onPostRespawn(PlayerPostRespawnEvent event) {
        manager.pendingRestore(event.getPlayer().getUniqueId()).ifPresent(backup -> {
            synchronizer.restore(event.getPlayer(), backup.state());
            manager.markRestored(backup);
        });
        if (!manager.isEnabled()) {
            return;
        }
        HealthState canonical = manager.current().orElseThrow();
        if (canonical.phase() == HealthPhase.DEAD) {
            canonical = manager.revive();
            deathWavePrimary = null;
            deathWaveSource = null;
        }
        synchronizer.applyToAll(canonical);
    }

    @EventHandler(priority = EventPriority.LOW)
    public void onTickEnd(ServerTickEndEvent event) {
        if (!manager.isEnabled()) {
            departingContributions.clear();
            primaryDeaths.clear();
            playerDamageEvents.clear();
            deathWavePrimary = null;
            deathWaveSource = null;
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
        if (observedPlayers.isEmpty()
            && departingContributions.isEmpty()
            && primaryDeaths.isEmpty()
            && manager.current()
                .map(state -> state.phase() == HealthPhase.ALIVE)
                .orElse(false)) {
            playerDamageEvents.clear();
            return;
        }
        var contributions = new ArrayList<>(departingContributions);
        contributions.addAll(synchronizer.observePlayers(observedPlayers));
        departingContributions.clear();
        PlayerDeathEvent primaryDeath = primaryDeaths.stream()
            .filter(death -> !death.isCancelled())
            .findFirst()
            .orElse(null);
        primaryDeaths.clear();
        if (primaryDeath != null) {
            deathWavePrimary = primaryDeath.getPlayer().getUniqueId();
            deathWaveSource = primaryDeath.getDamageSource();
        }
        HealthState previous = manager.current().orElseThrow();
        if (contributions.isEmpty()
            && primaryDeath == null
            && previous.phase() == HealthPhase.ALIVE) {
            playerDamageEvents.clear();
            return;
        }
        long previousRevision = previous.revision();
        HealthState canonical = manager.applyTick(
            List.copyOf(contributions), primaryDeath != null
        );
        if (canonical.phase() == HealthPhase.DEAD) {
            if (deathWaveSource == null) {
                playerDamageEvents.stream()
                    .filter(damage -> !damage.isCancelled())
                    .reduce((first, second) -> second)
                    .ifPresent(damage -> {
                        deathWavePrimary = new UUID(0, 0);
                        deathWaveSource = damage.getDamageSource();
                    });
            }
            if (deathWavePrimary != null && deathWaveSource != null) {
                deathCoordinator.killOtherPlayers(
                    deathWavePrimary,
                    deathWaveSource
                );
            } else {
                deathCoordinator.killOtherPlayersWithoutSource(new UUID(0, 0));
            }
        } else {
            deathWavePrimary = null;
            deathWaveSource = null;
            if (canonical.revision() != previousRevision) {
                synchronizer.applyToAll(canonical);
            } else if (!observedPlayers.isEmpty()) {
                synchronizer.applyToPlayers(observedPlayers, canonical);
            }
        }
        playerDamageEvents.clear();
    }

    public void resetReplicas() {
        synchronizer.clear();
        deathCoordinator.clear();
        departingContributions.clear();
        primaryDeaths.clear();
        playerDamageEvents.clear();
        dirtyPlayers.clear();
        ticksSinceAudit = 0;
        deathWavePrimary = null;
        deathWaveSource = null;
    }

    private void markDirty(org.bukkit.entity.Entity entity) {
        if (manager.isEnabled() && entity instanceof Player player) {
            dirtyPlayers.add(player.getUniqueId());
        }
    }
}
