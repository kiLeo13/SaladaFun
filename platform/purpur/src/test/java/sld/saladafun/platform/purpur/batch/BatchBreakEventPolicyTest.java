package sld.saladafun.platform.purpur.batch;

import org.bukkit.event.block.BlockBreakEvent;
import org.junit.jupiter.api.Test;
import sld.saladafun.batchbreaking.BatchBlockAction;

import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.verifyNoInteractions;

class BatchBreakEventPolicyTest {

    @Test
    void noDropsSuppressesItemsAndExperience() {
        BlockBreakEvent event = mock(BlockBreakEvent.class);

        BatchBreakEventPolicy.apply(BatchBlockAction.NO_DROPS, event);

        verify(event).setDropItems(false);
        verify(event).setExpToDrop(0);
    }

    @Test
    void otherActionsDoNotMutateTheEvent() {
        BlockBreakEvent event = mock(BlockBreakEvent.class);

        BatchBreakEventPolicy.apply(BatchBlockAction.PLAYER_TOOL, event);

        verifyNoInteractions(event);
    }
}
