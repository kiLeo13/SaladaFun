package sld.saladafun.batchbreaking;

/**
 * Controls whether additional player-aware breaks consume tool durability.
 */
public enum ToolDurabilityMode {
    /**
     * Only the original, ordinary block break may consume durability.
     */
    SINGLE_USE,

    /**
     * Every additional player-aware block break may consume durability.
     */
    PER_BLOCK
}
