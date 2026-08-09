package sld.saladafun.platform.purpur.shared.health;

import com.destroystokyo.paper.event.server.ServerTickEndEvent;
import org.bukkit.damage.DamageSource;
import org.bukkit.entity.Player;
import org.bukkit.event.entity.EntityDamageEvent;
import org.bukkit.event.entity.PlayerDeathEvent;
import org.junit.jupiter.api.Test;
import sld.saladafun.shared.health.HealthPhase;
import sld.saladafun.shared.health.HealthContribution;
import sld.saladafun.shared.health.HealthState;
import sld.saladafun.shared.health.SharedHealthManager;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

import static org.mockito.ArgumentMatchers.anyList;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.times;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class SharedHealthHandlerTest {

    @Test
    void idleTickDoesNotPollPlayersOrTouchTheAggregate() {
        SharedHealthManager manager = mock(SharedHealthManager.class);
        PlayerHealthSynchronizer synchronizer = mock(PlayerHealthSynchronizer.class);
        when(manager.isEnabled()).thenReturn(true);
        when(manager.current()).thenReturn(Optional.of(new HealthState(
            20.0, 20.0, 0.0, 0.0, HealthPhase.ALIVE, 0
        )));
        var handler = new SharedHealthHandler(
            manager,
            mock(PurpurHealthMapper.class),
            synchronizer,
            mock(SharedDeathCoordinator.class),
            20
        );

        handler.onTickEnd(mock(ServerTickEndEvent.class));

        verify(synchronizer, never()).observePlayers(
            org.mockito.ArgumentMatchers.anyCollection()
        );
        verify(manager, never()).applyTick(
            anyList(),
            org.mockito.ArgumentMatchers.anyBoolean()
        );
    }

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
        HealthState alive = new HealthState(
            20.0, 20.0, 0.0, 10.0, HealthPhase.ALIVE, 0
        );
        when(deaths.consumeGeneratedDeath(playerId)).thenReturn(false);
        when(synchronizer.observeOnline()).thenReturn(List.of());
        HealthState dead = new HealthState(
            0.0, 20.0, 0.0, 10.0, HealthPhase.DEAD, 1
        );
        when(manager.current()).thenReturn(
            Optional.of(alive),
            Optional.of(dead),
            Optional.of(dead)
        );
        when(manager.applyTick(anyList(), org.mockito.ArgumentMatchers.eq(true)))
            .thenReturn(dead);
        when(manager.applyTick(anyList(), org.mockito.ArgumentMatchers.eq(false)))
            .thenReturn(dead);
        var handler = new SharedHealthHandler(
            manager, mapper, synchronizer, deaths, 20
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
        when(manager.current()).thenReturn(Optional.of(alive));
        when(manager.applyTick(anyList(), org.mockito.ArgumentMatchers.eq(false)))
            .thenReturn(alive);
        var handler = new SharedHealthHandler(
            manager,
            mock(PurpurHealthMapper.class),
            synchronizer,
            deaths,
            20
        );

        handler.onDeath(deathEvent);
        handler.onTickEnd(mock(ServerTickEndEvent.class));

        verify(manager, never()).applyTick(
            anyList(),
            org.mockito.ArgumentMatchers.anyBoolean()
        );
        verify(deaths, never()).killOtherPlayers(
            org.mockito.ArgumentMatchers.any(UUID.class),
            org.mockito.ArgumentMatchers.any(DamageSource.class)
        );
    }

    @Test
    void aggregateDeathUsesTheLastAcceptedDamageSourceForEveryPlayer() {
        SharedHealthManager manager = mock(SharedHealthManager.class);
        PlayerHealthSynchronizer synchronizer = mock(PlayerHealthSynchronizer.class);
        SharedDeathCoordinator deaths = mock(SharedDeathCoordinator.class);
        Player player = mock(Player.class);
        EntityDamageEvent firstDamage = mock(EntityDamageEvent.class);
        EntityDamageEvent lastDamage = mock(EntityDamageEvent.class);
        DamageSource firstSource = mock(DamageSource.class);
        DamageSource lastSource = mock(DamageSource.class);
        when(player.getUniqueId()).thenReturn(UUID.randomUUID());
        when(firstDamage.getEntity()).thenReturn(player);
        when(firstDamage.getDamageSource()).thenReturn(firstSource);
        when(lastDamage.getEntity()).thenReturn(player);
        when(lastDamage.getDamageSource()).thenReturn(lastSource);
        when(manager.isEnabled()).thenReturn(true);
        HealthState alive = new HealthState(
            2.0, 20.0, 0.0, 10.0, HealthPhase.ALIVE, 0
        );
        HealthState dead = new HealthState(
            0.0, 20.0, 0.0, 10.0, HealthPhase.DEAD, 1
        );
        when(manager.current()).thenReturn(Optional.of(alive));
        UUID actorId = player.getUniqueId();
        HealthContribution contribution = new HealthContribution(
            actorId, -2.0, 0.0, false, 20.0, 10.0
        );
        when(synchronizer.observePlayers(
            org.mockito.ArgumentMatchers.anyCollection()
        )).thenReturn(List.of(contribution));
        when(manager.applyTick(anyList(), org.mockito.ArgumentMatchers.eq(false)))
            .thenReturn(dead);
        when(synchronizer.observeOnline()).thenReturn(List.of());
        var handler = new SharedHealthHandler(
            manager,
            mock(PurpurHealthMapper.class),
            synchronizer,
            deaths,
            20
        );

        handler.onDamage(firstDamage);
        handler.onDamage(lastDamage);
        handler.onTickEnd(mock(ServerTickEndEvent.class));

        verify(deaths).killOtherPlayers(new UUID(0, 0), lastSource);
        verify(deaths, never()).killOtherPlayers(
            org.mockito.ArgumentMatchers.any(UUID.class),
            org.mockito.ArgumentMatchers.same(firstSource)
        );
    }
}
