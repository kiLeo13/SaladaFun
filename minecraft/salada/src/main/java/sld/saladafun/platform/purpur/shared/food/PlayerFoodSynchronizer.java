package sld.saladafun.platform.purpur.shared.food;

import org.bukkit.Server;
import org.bukkit.entity.Player;
import sld.saladafun.shared.food.FoodContribution;
import sld.saladafun.shared.food.FoodState;

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

/** Applies canonical food and tracks the exact replica last written to each player. */
public final class PlayerFoodSynchronizer {
    private final Server server;
    private final PurpurFoodMapper mapper;
    private final Map<UUID, FoodState> replicas = new HashMap<>();
    private final Set<UUID> synchronizing = new HashSet<>();

    public PlayerFoodSynchronizer(Server server, PurpurFoodMapper mapper) {
        this.server = Objects.requireNonNull(server, "server");
        this.mapper = Objects.requireNonNull(mapper, "mapper");
    }

    public void apply(Player player, FoodState canonical) {
        UUID playerId = player.getUniqueId();
        synchronizing.add(playerId);
        try {
            mapper.apply(player, canonical);
            replicas.put(playerId, mapper.snapshot(player, canonical.revision()));
        } finally {
            synchronizing.remove(playerId);
        }
    }

    public void applyToAll(FoodState canonical) {
        for (Player player : server.getOnlinePlayers()) {
            apply(player, canonical);
        }
    }

    public List<UUID> onlinePlayerIds() {
        return server.getOnlinePlayers().stream().map(Player::getUniqueId).toList();
    }

    public void applyToPlayers(Collection<UUID> playerIds, FoodState canonical) {
        for (UUID playerId : playerIds) {
            Player player = server.getPlayer(playerId);
            if (player != null) {
                apply(player, canonical);
            }
        }
    }

    public List<FoodContribution> observeOnline() {
        var contributions = new ArrayList<FoodContribution>();
        for (Player player : server.getOnlinePlayers()) {
            observe(player).ifPresent(contributions::add);
        }
        return List.copyOf(contributions);
    }

    public List<FoodContribution> observePlayers(Collection<UUID> playerIds) {
        var contributions = new ArrayList<FoodContribution>();
        for (UUID playerId : playerIds) {
            Player player = server.getPlayer(playerId);
            if (player != null) {
                observe(player).ifPresent(contributions::add);
            }
        }
        return List.copyOf(contributions);
    }

    public Optional<FoodContribution> observe(Player player) {
        UUID playerId = player.getUniqueId();
        if (synchronizing.contains(playerId)) {
            return Optional.empty();
        }
        FoodState replica = replicas.get(playerId);
        if (replica == null) {
            return Optional.empty();
        }
        FoodState observed = mapper.snapshot(player, replica.revision());
        replicas.put(playerId, observed);
        return Optional.of(new FoodContribution(
            playerId,
            observed.foodLevel() - replica.foodLevel(),
            observed.saturation() - replica.saturation(),
            observed.exhaustion() - replica.exhaustion()
        ));
    }

    public void forget(UUID playerId) {
        replicas.remove(playerId);
        synchronizing.remove(playerId);
    }

    public void clear() {
        replicas.clear();
        synchronizing.clear();
    }
}
