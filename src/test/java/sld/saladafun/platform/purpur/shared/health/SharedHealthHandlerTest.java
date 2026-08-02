package sld.saladafun.platform.purpur.shared.health;

import com.destroystokyo.paper.event.server.ServerTickEndEvent;
import org.bukkit.damage.DamageSource;
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
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.times;
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
        DamageSource damageSource = mock(DamageSource.class);
        when(player.getUniqueId()).thenReturn(playerId);
        when(deathEvent.getPlayer()).thenReturn(player);
        when(deathEvent.getDamageSource()).thenReturn(damageSource);
        when(manager.isEnabled()).thenReturn(true);
        when(deaths.consumeGeneratedDeath(playerId)).thenReturn(false);
        when(synchronizer.observeOnline()).thenReturn(List.of());
        HealthState dead = new HealthState(
            0.0, 20.0, 0.0, 10.0, HealthPhase.DEAD, 1
        );
        when(manager.applyTick(anyList(), org.mockito.ArgumentMatchers.eq(true)))
            .thenReturn(dead);
        when(manager.applyTick(anyList(), org.mockito.ArgumentMatchers.eq(false)))
            .thenReturn(dead);
        var handler = new SharedHealthHandler(
            manager, mapper, synchronizer, deaths
        );

        handler.onDeath(deathEvent);
        handler.onTickEnd(mock(ServerTickEndEvent.class));
        handler.onTickEnd(mock(ServerTickEndEvent.class));

        verify(manager).applyTick(List.of(), true);
        verify(manager).applyTick(List.of(), false);
        verify(deaths, times(2)).killOtherPlayers(playerId, damageSource);
    }

    @Test
    void cancelledPrimaryDeathDoesNotStartASharedWave() {
        SharedHealthManager manager = mock(SharedHealthManager.class);
        PlayerHealthSynchronizer synchronizer = mock(PlayerHealthSynchronizer.class);
        SharedDeathCoordinator deaths = mock(SharedDeathCoordinator.class);
        Player player = mock(Player.class);
        PlayerDeathEvent deathEvent = mock(PlayerDeathEvent.class);
        when(player.getUniqueId()).thenReturn(UUID.randomUUID());
        when(deathEvent.getPlayer()).thenReturn(player);
        when(deathEvent.isCancelled()).thenReturn(true);
        when(manager.isEnabled()).thenReturn(true);
        when(synchronizer.observeOnline()).thenReturn(List.of());
        HealthState alive = new HealthState(
            20.0, 20.0, 0.0, 10.0, HealthPhase.ALIVE, 0
        );
        when(manager.applyTick(anyList(), org.mockito.ArgumentMatchers.eq(false)))
            .thenReturn(alive);
        var handler = new SharedHealthHandler(
            manager,
            mock(PurpurHealthMapper.class),
            synchronizer,
            deaths
        );

        handler.onDeath(deathEvent);
        handler.onTickEnd(mock(ServerTickEndEvent.class));

        verify(manager).applyTick(List.of(), false);
        verify(deaths, never()).killOtherPlayers(
            org.mockito.ArgumentMatchers.any(UUID.class),
            org.mockito.ArgumentMatchers.any(DamageSource.class)
        );
    }
}
