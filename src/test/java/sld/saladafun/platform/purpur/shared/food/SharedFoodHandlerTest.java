package sld.saladafun.platform.purpur.shared.food;

import com.destroystokyo.paper.event.server.ServerTickEndEvent;
import org.junit.jupiter.api.Test;
import sld.saladafun.shared.food.SharedFoodManager;

import static org.mockito.ArgumentMatchers.anyCollection;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class SharedFoodHandlerTest {

    @Test
    void idleTickDoesNotPollPlayersOrTouchTheAggregate() {
        SharedFoodManager manager = mock(SharedFoodManager.class);
        PlayerFoodSynchronizer synchronizer = mock(PlayerFoodSynchronizer.class);
        when(manager.isEnabled()).thenReturn(true);
        var handler = new SharedFoodHandler(
            manager,
            mock(PurpurFoodMapper.class),
            synchronizer,
            20
        );

        handler.onTickEnd(mock(ServerTickEndEvent.class));

        verify(synchronizer, never()).observePlayers(anyCollection());
        verify(manager, never()).applyTick(org.mockito.ArgumentMatchers.anyList());
    }
}
