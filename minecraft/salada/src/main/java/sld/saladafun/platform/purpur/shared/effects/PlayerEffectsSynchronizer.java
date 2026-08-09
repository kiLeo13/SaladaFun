package sld.saladafun.platform.purpur.shared.effects;

import org.bukkit.Server;
import org.bukkit.entity.Player;
import sld.saladafun.shared.effects.EffectChange;
import sld.saladafun.shared.effects.EffectState;
import sld.saladafun.shared.effects.EffectsState;

import java.util.ArrayList;
import java.util.Collection;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Optional;
import java.util.Set;
import java.util.UUID;

/** Applies canonical effects and observes only event-dirtied effect types. */
public final class PlayerEffectsSynchronizer {
    private final Server server;
    private final PurpurEffectsMapper mapper;
    private final Map<UUID, EffectsState> replicas = new HashMap<>();
    private final Set<UUID> synchronizing = new HashSet<>();

    public PlayerEffectsSynchronizer(Server server, PurpurEffectsMapper mapper) {
        this.server = Objects.requireNonNull(server, "server");
        this.mapper = Objects.requireNonNull(mapper, "mapper");
    }

    public void apply(Player player, EffectsState canonical) {
        UUID playerId = player.getUniqueId();
        synchronizing.add(playerId);
        try {
            mapper.apply(player, canonical);
            replicas.put(
                playerId,
                mapper.snapshot(player, canonical.revision())
            );
        } finally {
            synchronizing.remove(playerId);
        }
    }

    public void applyToAll(EffectsState canonical) {
        for (Player player : server.getOnlinePlayers()) {
            apply(player, canonical);
        }
    }

    public void restore(Player player, EffectsState personalState) {
        apply(player, personalState);
        replicas.remove(player.getUniqueId());
    }

    public List<EffectChange> observe(
        Player player,
        Collection<String> explicitTypes,
        boolean audit
    ) {
        UUID playerId = player.getUniqueId();
        EffectsState previous = replicas.get(playerId);
        if (previous == null || synchronizing.contains(playerId)) {
            return List.of();
        }
        EffectsState observed = mapper.snapshot(player, previous.revision());
        Set<String> types = new HashSet<>(explicitTypes);
        if (audit) {
            types.addAll(previous.effects().keySet());
            types.addAll(observed.effects().keySet());
        }
        List<EffectChange> changes = new ArrayList<>();
        for (String type : types) {
            EffectState before = previous.effects().get(type);
            EffectState after = observed.effects().get(type);
            if (Objects.equals(before, after)) {
                continue;
            }
            if (audit
                && !explicitTypes.contains(type)
                && isNaturalCountdown(before, after)) {
                continue;
            }
            changes.add(after == null
                ? EffectChange.remove(playerId, type)
                : EffectChange.replace(playerId, after));
        }
        replicas.put(playerId, observed);
        return List.copyOf(changes);
    }

    public Optional<Map<String, EffectState>> firstReplica() {
        return replicas.values().stream().findFirst().map(EffectsState::effects);
    }

    public List<UUID> onlinePlayerIds() {
        return server.getOnlinePlayers().stream().map(Player::getUniqueId).toList();
    }

    public Player onlinePlayer(UUID playerId) {
        return server.getPlayer(playerId);
    }

    public boolean isSynchronizing(UUID playerId) {
        return synchronizing.contains(playerId);
    }

    public void forget(UUID playerId) {
        replicas.remove(playerId);
        synchronizing.remove(playerId);
    }

    public void clear() {
        replicas.clear();
        synchronizing.clear();
    }

    private boolean isNaturalCountdown(EffectState before, EffectState after) {
        if (before == null || after == null || !before.sameDefinition(after)) {
            return false;
        }
        if (before.durationTicks() == EffectState.INFINITE_DURATION) {
            return after.durationTicks() == EffectState.INFINITE_DURATION;
        }
        return after.durationTicks() <= before.durationTicks();
    }
}
