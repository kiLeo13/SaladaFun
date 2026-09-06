package sld.saladafun.platform.purpur.shared.health;

import org.bukkit.Server;
import org.bukkit.entity.Player;
import org.junit.jupiter.api.Test;
import sld.saladafun.shared.health.HealthPhase;
import sld.saladafun.shared.health.HealthState;

import java.util.UUID;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

class PlayerHealthSynchronizerTest {

    @Test
    void observesNaturalRangeChangesWithoutFeedingBackTheSharedOverride() {
        UUID playerId = UUID.randomUUID();
        Player player = mock(Player.class);
        when(player.getUniqueId()).thenReturn(playerId);
        PurpurHealthMapper mapper = mock(PurpurHealthMapper.class);
        HealthState canonical = state(20.0, 20.0);
        when(mapper.snapshot(player, 0)).thenReturn(canonical);
        when(mapper.naturalMaximumHealth(player)).thenReturn(20.0, 4.0, 4.0);
        when(mapper.naturalMaximumAbsorption(player)).thenReturn(0.0);
        PlayerHealthSynchronizer synchronizer = new PlayerHealthSynchronizer(
            mock(Server.class), mapper
        );
        synchronizer.apply(player, canonical);

        var changed = synchronizer.observe(player).orElseThrow();
        var stable = synchronizer.observe(player).orElseThrow();

        assertTrue(changed.rangeChanged());
        assertEquals(4.0, changed.observedMaximumHealth());
        assertFalse(stable.rangeChanged());
    }

    private HealthState state(double health, double maximumHealth) {
        return new HealthState(
            health,
            maximumHealth,
            0.0,
            0.0,
            HealthPhase.ALIVE,
            0
        );
    }
}
