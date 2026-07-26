package sld.saladafun.batchbreaking;

/**
 * Platform-independent batch-breaking scope.
 */
public sealed interface BatchBreakingSetting
    permits BatchBreakingSetting.Disabled, BatchBreakingSetting.AllSentChunks,
    BatchBreakingSetting.CubicRadius {

    record Disabled() implements BatchBreakingSetting {
    }

    record AllSentChunks() implements BatchBreakingSetting {
    }

    record CubicRadius(int blocks) implements BatchBreakingSetting {
        public CubicRadius {
            if (blocks < 1) {
                throw new IllegalArgumentException("Cubic radius must be positive");
            }
        }
    }
}
