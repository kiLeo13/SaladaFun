package sld.saladafun.platform.purpur.shared.health;

import org.bukkit.Server;
import org.bukkit.entity.Player;

import java.util.HashSet;
import java.util.Objects;
import java.util.Set;
import java.util.UUID;

/** Guards the forced secondary deaths belonging to one shared-health death wave. */
public final class SharedDeathCoordinator {
    private final Server server;
    private final Set<UUID> generatedDeaths = new HashSet<>();

    public SharedDeathCoordinator(Server server) {
        this.server = Objects.requireNonNull(server, "server");
    }

    public void killOtherPlayers(UUID primaryPlayerId) {
        for (Player player : server.getOnlinePlayers()) {
            UUID playerId = player.getUniqueId();
            if (playerId.equals(primaryPlayerId) || player.isDead()) {
                continue;
            }
            generatedDeaths.add(playerId);
            player.setHealth(0.0);
        }
    }

    public boolean consumeGeneratedDeath(UUID playerId) {
        return generatedDeaths.remove(playerId);
    }

    public void clear() {
        generatedDeaths.clear();
    }
}
