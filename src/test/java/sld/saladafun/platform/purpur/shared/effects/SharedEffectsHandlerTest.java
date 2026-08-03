package sld.saladafun.platform.purpur.shared.effects;

import com.destroystokyo.paper.event.server.ServerTickEndEvent;
import org.junit.jupiter.api.Test;
import sld.saladafun.shared.effects.SharedEffectsManager;

import static org.mockito.ArgumentMatchers.anyCollection;
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
}
