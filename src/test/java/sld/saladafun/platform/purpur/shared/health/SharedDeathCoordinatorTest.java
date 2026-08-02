package sld.saladafun.platform.purpur.shared.health;

import org.bukkit.Server;
import org.bukkit.damage.DamageSource;
import org.bukkit.entity.Player;
import org.junit.jupiter.api.Test;

import java.util.Set;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.mockito.Mockito.doReturn;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class SharedDeathCoordinatorTest {

    @Test
    void appliesThePrimaryDamageSourceToEverySecondaryDeath() {
        UUID primaryId = UUID.randomUUID();
        UUID secondaryId = UUID.randomUUID();
        Player primary = player(primaryId);
        Player secondary = player(secondaryId);
        when(secondary.isDead()).thenReturn(false, true);
        Server server = mock(Server.class);
        doReturn(Set.of(primary, secondary)).when(server).getOnlinePlayers();
        DamageSource source = mock(DamageSource.class);
        var coordinator = new SharedDeathCoordinator(server);

        coordinator.killOtherPlayers(primaryId, source);

        verify(primary, never()).damage(
            org.mockito.ArgumentMatchers.anyDouble(),
            org.mockito.ArgumentMatchers.any(DamageSource.class)
        );
        verify(secondary).damage((double) Float.MAX_VALUE, source);
        assertTrue(coordinator.consumeGeneratedDeath(secondaryId));
        assertFalse(coordinator.consumeGeneratedDeath(secondaryId));
    }

    private Player player(UUID id) {
        Player player = mock(Player.class);
        when(player.getUniqueId()).thenReturn(id);
        return player;
    }
}
