package sld.saladafun.platform.purpur.shared.effects;

import com.destroystokyo.paper.event.server.ServerTickEndEvent;
import org.bukkit.entity.Player;
import org.bukkit.event.player.PlayerQuitEvent;
import org.junit.jupiter.api.Test;
import sld.saladafun.shared.effects.SharedEffectsManager;

import java.util.List;
import java.util.Set;
import java.util.UUID;

import static org.mockito.ArgumentMatchers.anyCollection;
import static org.mockito.Mockito.eq;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class SharedEffectsHandlerTest {

    @Test
    void idleTickDoesNotPollPlayersOrTouchTheAggregate() {
        SharedEffectsManager manager = mock(SharedEffectsManager.class);
        PlayerEffectsSynchronizer synchronizer = mock(
            PlayerEffectsSynchronizer.class
        );
        when(manager.isEnabled()).thenReturn(true);
        var handler = new SharedEffectsHandler(
            manager,
            mock(PurpurEffectsMapper.class),
            synchronizer,
            20
        );

        handler.onTickEnd(mock(ServerTickEndEvent.class));

        verify(synchronizer, never()).observe(
            org.mockito.ArgumentMatchers.any(),
            anyCollection(),
            org.mockito.ArgumentMatchers.anyBoolean()
        );
        verify(manager, never()).applyTick(
            org.mockito.ArgumentMatchers.anyList()
        );
    }

    @Test
    void quittingPlayerDoesNotTurnNaturalCountdownIntoAnExplicitChange() {
        SharedEffectsManager manager = mock(SharedEffectsManager.class);
        PurpurEffectsMapper mapper = mock(PurpurEffectsMapper.class);
        PlayerEffectsSynchronizer synchronizer = mock(
            PlayerEffectsSynchronizer.class
        );
        Player player = mock(Player.class);
        PlayerQuitEvent event = mock(PlayerQuitEvent.class);
        UUID playerId = UUID.randomUUID();
        when(manager.isEnabled()).thenReturn(true);
        when(player.getUniqueId()).thenReturn(playerId);
        when(event.getPlayer()).thenReturn(player);
        when(synchronizer.observe(player, Set.of(), true))
            .thenReturn(List.of());
        var handler = new SharedEffectsHandler(
            manager,
            mapper,
            synchronizer,
            20
        );

        handler.onQuit(event);

        verify(synchronizer).observe(player, Set.of(), true);
        verify(mapper, never()).snapshot(eq(player),
            org.mockito.ArgumentMatchers.anyLong());
        verify(synchronizer).forget(playerId);
    }
}
