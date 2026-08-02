package sld.saladafun.platform.purpur.shared.health;

import com.destroystokyo.paper.event.server.ServerTickEndEvent;
import org.bukkit.entity.Player;
import org.bukkit.event.entity.PlayerDeathEvent;
import org.junit.jupiter.api.Test;
import sld.saladafun.shared.health.HealthPhase;
import sld.saladafun.shared.health.HealthState;
import sld.saladafun.shared.health.SharedHealthManager;

import java.util.List;
import java.util.UUID;

import static org.mockito.ArgumentMatchers.anyList;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class SharedHealthHandlerTest {

    @Test
    void genuineDeathMakesTheTickLethalAndStartsOneWave() {
        SharedHealthManager manager = mock(SharedHealthManager.class);
        PurpurHealthMapper mapper = mock(PurpurHealthMapper.class);
        PlayerHealthSynchronizer synchronizer = mock(PlayerHealthSynchronizer.class);
        SharedDeathCoordinator deaths = mock(SharedDeathCoordinator.class);
        UUID playerId = UUID.randomUUID();
        Player player = mock(Player.class);
        PlayerDeathEvent deathEvent = mock(PlayerDeathEvent.class);
        when(player.getUniqueId()).thenReturn(playerId);
        when(deathEvent.getPlayer()).thenReturn(player);
        when(manager.isEnabled()).thenReturn(true);
        when(deaths.consumeGeneratedDeath(playerId)).thenReturn(false);
        when(synchronizer.observeOnline()).thenReturn(List.of());
        HealthState dead = new HealthState(
            0.0, 20.0, 0.0, 10.0, HealthPhase.DEAD, 1
        );
        when(manager.applyTick(anyList(), org.mockito.ArgumentMatchers.eq(true)))
            .thenReturn(dead);
        var handler = new SharedHealthHandler(
            manager, mapper, synchronizer, deaths
        );

        handler.onDeath(deathEvent);
        handler.onTickEnd(mock(ServerTickEndEvent.class));

        verify(manager).applyTick(List.of(), true);
        verify(deaths).killOtherPlayers(playerId);
    }
}
