package sld.saladafun.platform.purpur.shared.effects;

import com.destroystokyo.paper.event.player.PlayerPostRespawnEvent;
import com.destroystokyo.paper.event.server.ServerTickEndEvent;
import org.bukkit.entity.Player;
import org.bukkit.event.EventHandler;
import org.bukkit.event.EventPriority;
import org.bukkit.event.Listener;
import org.bukkit.event.entity.EntityPotionEffectEvent;
import org.bukkit.event.player.PlayerJoinEvent;
import org.bukkit.event.player.PlayerQuitEvent;
import sld.saladafun.shared.effects.EffectChange;
import sld.saladafun.shared.effects.EffectsState;
import sld.saladafun.shared.effects.SharedEffectsManager;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Set;
import java.util.UUID;

/** Event-driven Purpur adapter for canonical active potion effects. */
public final class SharedEffectsHandler implements Listener {
    private final SharedEffectsManager manager;
    private final PurpurEffectsMapper mapper;
    private final PlayerEffectsSynchronizer synchronizer;
    private final int safetyAuditIntervalTicks;
    private final List<EntityPotionEffectEvent> pendingEvents = new ArrayList<>();
    private final List<EffectChange> departingChanges = new ArrayList<>();
    private int ticksSinceAudit;

    public SharedEffectsHandler(
        SharedEffectsManager manager,
        PurpurEffectsMapper mapper,
        PlayerEffectsSynchronizer synchronizer,
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
            synchronizer.restore(player, backup.state());
            manager.markRestored(backup);
        });
        if (!manager.isEnabled()) {
            synchronizer.forget(player.getUniqueId());
            return;
        }
        EffectsState personal = mapper.snapshot(player, 0);
        EffectsState canonical = manager.join(player.getUniqueId(), personal);
        synchronizer.apply(player, canonical);
    }

    @EventHandler(priority = EventPriority.LOWEST)
    public void onPotionEffect(EntityPotionEffectEvent event) {
        if (!manager.isEnabled() || !(event.getEntity() instanceof Player player)) {
            return;
        }
        if (!synchronizer.isSynchronizing(player.getUniqueId())) {
            pendingEvents.add(event);
        }
    }

    @EventHandler(priority = EventPriority.LOWEST)
    public void onQuit(PlayerQuitEvent event) {
        Player player = event.getPlayer();
        if (manager.isEnabled()) {
            Set<String> types = new HashSet<>(
                mapper.snapshot(player, 0).effects().keySet()
            );
            pendingEvents.stream()
                .filter(candidate -> !candidate.isCancelled())
                .filter(candidate -> candidate.getEntity().getUniqueId()
                    .equals(player.getUniqueId()))
                .map(candidate -> candidate.getModifiedType().getKey().asString())
                .forEach(types::add);
            departingChanges.addAll(synchronizer.observe(
                player,
                types,
                true
            ));
        }
        pendingEvents.removeIf(candidate ->
            candidate.getEntity().getUniqueId().equals(player.getUniqueId())
        );
        synchronizer.forget(player.getUniqueId());
    }

    @EventHandler(priority = EventPriority.LOWEST)
    public void onPostRespawn(PlayerPostRespawnEvent event) {
        reconcilePlayer(event.getPlayer());
    }

    @EventHandler(priority = EventPriority.LOW)
    public void onTickEnd(ServerTickEndEvent event) {
        if (!manager.isEnabled()) {
            resetTransientState();
            return;
        }
        ticksSinceAudit++;
        boolean audit = ticksSinceAudit >= safetyAuditIntervalTicks;
        if (audit) {
            ticksSinceAudit = 0;
        }
        Map<UUID, Set<String>> explicitTypes = acceptedTypes();
        pendingEvents.clear();
        if (!audit && explicitTypes.isEmpty() && departingChanges.isEmpty()) {
            return;
        }
        Set<UUID> observedPlayers = new HashSet<>(explicitTypes.keySet());
        if (audit) {
            observedPlayers.addAll(synchronizer.onlinePlayerIds());
        }
        List<EffectChange> changes = new ArrayList<>(departingChanges);
        departingChanges.clear();
        for (UUID playerId : observedPlayers) {
            Player player = synchronizer.onlinePlayer(playerId);
            if (player == null) {
                continue;
            }
            changes.addAll(synchronizer.observe(
                player,
                explicitTypes.getOrDefault(playerId, Set.of()),
                audit
            ));
        }
        if (audit) {
            synchronizer.firstReplica().ifPresent(manager::refreshDurations);
        }
        if (changes.isEmpty()) {
            return;
        }
        long previousRevision = manager.current().orElseThrow().revision();
        EffectsState canonical = manager.applyTick(List.copyOf(changes));
        if (canonical.revision() != previousRevision) {
            synchronizer.applyToAll(canonical);
        }
    }

    public void resetReplicas() {
        synchronizer.clear();
        resetTransientState();
    }

    private Map<UUID, Set<String>> acceptedTypes() {
        Map<UUID, Set<String>> types = new HashMap<>();
        for (EntityPotionEffectEvent event : pendingEvents) {
            if (event.isCancelled()) {
                continue;
            }
            UUID playerId = event.getEntity().getUniqueId();
            types.computeIfAbsent(playerId, ignored -> new HashSet<>())
                .add(event.getModifiedType().getKey().asString());
        }
        return types;
    }

    private void resetTransientState() {
        pendingEvents.clear();
        departingChanges.clear();
        ticksSinceAudit = 0;
    }
}
