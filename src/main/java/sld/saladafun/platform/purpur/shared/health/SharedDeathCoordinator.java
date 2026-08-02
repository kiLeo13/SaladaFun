package sld.saladafun.platform.purpur.shared.health;

import org.bukkit.Server;
import org.bukkit.damage.DamageSource;
import org.bukkit.entity.Player;

import java.util.HashSet;
import java.util.Objects;
import java.util.Set;
import java.util.UUID;

/** Guards the forced secondary deaths belonging to one shared-health death wave. */
public final class SharedDeathCoordinator {
    private static final double LETHAL_DAMAGE = Float.MAX_VALUE;

    private final Server server;
    private final Set<UUID> generatedDeaths = new HashSet<>();

    public SharedDeathCoordinator(Server server) {
        this.server = Objects.requireNonNull(server, "server");
    }

    public void killOtherPlayers(
        UUID primaryPlayerId,
        DamageSource primaryDamageSource
    ) {
        Objects.requireNonNull(primaryPlayerId, "primaryPlayerId");
        Objects.requireNonNull(primaryDamageSource, "primaryDamageSource");
        for (Player player : server.getOnlinePlayers()) {
            UUID playerId = player.getUniqueId();
            if (playerId.equals(primaryPlayerId) || player.isDead()) {
                continue;
            }
            generatedDeaths.add(playerId);
            player.damage(LETHAL_DAMAGE, primaryDamageSource);
            if (!player.isDead()) {
                generatedDeaths.add(playerId);
                player.setHealth(0.0);
            }
            if (!player.isDead()) {
                generatedDeaths.remove(playerId);
            }
        }
    }

    /** Kills a synthetic zero-pool wave when no real death supplied a source. */
    public void killOtherPlayersWithoutSource(UUID primaryPlayerId) {
        Objects.requireNonNull(primaryPlayerId, "primaryPlayerId");
        for (Player player : server.getOnlinePlayers()) {
            UUID playerId = player.getUniqueId();
            if (playerId.equals(primaryPlayerId) || player.isDead()) {
                continue;
            }
            generatedDeaths.add(playerId);
            player.setHealth(0.0);
            if (!player.isDead()) {
                generatedDeaths.remove(playerId);
            }
        }
    }

    public boolean consumeGeneratedDeath(UUID playerId) {
        return generatedDeaths.remove(playerId);
    }

    public void clear() {
        generatedDeaths.clear();
    }
}
