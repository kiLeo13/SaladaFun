package sld.saladafun.platform.purpur.batch;

import org.bukkit.event.block.BlockBreakEvent;
import sld.saladafun.batchbreaking.BatchBlockAction;

import java.util.Objects;

/**
 * Applies cooperative event mutations for a generated batch break.
 */
final class BatchBreakEventPolicy {
    private BatchBreakEventPolicy() {
    }

    static void apply(BatchBlockAction action, BlockBreakEvent event) {
        Objects.requireNonNull(action, "action");
        Objects.requireNonNull(event, "event");

        if (action == BatchBlockAction.NO_DROPS) {
            event.setDropItems(false);
            event.setExpToDrop(0);
        }
    }
}
