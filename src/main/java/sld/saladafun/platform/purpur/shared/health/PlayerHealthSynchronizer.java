package sld.saladafun.platform.purpur.shared.health;

import org.bukkit.Server;
import org.bukkit.entity.Player;
import sld.saladafun.shared.health.HealthContribution;
import sld.saladafun.shared.health.HealthState;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Set;
import java.util.UUID;

/** Applies canonical health and tracks the exact replica last written to each player. */
public final class PlayerHealthSynchronizer {
    private static final double EPSILON = 1.0E-6;

    private final Server server;
    private final PurpurHealthMapper mapper;
    private final Map<UUID, HealthState> replicas = new HashMap<>();
    private final Set<UUID> synchronizing = new HashSet<>();

    public PlayerHealthSynchronizer(Server server, PurpurHealthMapper mapper) {
        this.server = Objects.requireNonNull(server, "server");
        this.mapper = Objects.requireNonNull(mapper, "mapper");
    }

    public void apply(Player player, HealthState canonical) {
        UUID playerId = player.getUniqueId();
        synchronizing.add(playerId);
        try {
            mapper.apply(player, canonical);
            replicas.put(playerId, mapper.snapshot(player, canonical.revision()));
        } finally {
            synchronizing.remove(playerId);
        }
    }

    public void applyToAll(HealthState canonical) {
        for (Player player : server.getOnlinePlayers()) {
            apply(player, canonical);
        }
    }

    public List<HealthContribution> observeOnline() {
        var contributions = new ArrayList<HealthContribution>();
        for (Player player : server.getOnlinePlayers()) {
            observe(player).ifPresent(contributions::add);
        }
        return List.copyOf(contributions);
    }

    public java.util.Optional<HealthContribution> observe(Player player) {
        UUID playerId = player.getUniqueId();
        if (synchronizing.contains(playerId)) {
            return java.util.Optional.empty();
        }
        HealthState replica = replicas.get(playerId);
        if (replica == null) {
            return java.util.Optional.empty();
        }
        HealthState observed = mapper.snapshot(player, replica.revision());
        boolean rangeChanged = differs(
            observed.maximumHealth(), replica.maximumHealth()
        ) || differs(observed.maximumAbsorption(), replica.maximumAbsorption());
        HealthContribution contribution = new HealthContribution(
            playerId,
            observed.health() - replica.health(),
            observed.absorption() - replica.absorption(),
            rangeChanged,
            observed.maximumHealth(),
            observed.maximumAbsorption()
        );
        replicas.put(playerId, observed);
        return java.util.Optional.of(contribution);
    }

    public void forget(UUID playerId) {
        replicas.remove(playerId);
        synchronizing.remove(playerId);
    }

    public void clear() {
        replicas.clear();
        synchronizing.clear();
    }

    private boolean differs(double left, double right) {
        return Math.abs(left - right) > EPSILON;
    }
}
