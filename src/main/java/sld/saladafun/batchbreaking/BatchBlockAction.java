package sld.saladafun.batchbreaking;

/**
 * Defines how each additional block discovered by a batch job is destroyed.
 */
public enum BatchBlockAction {
    /**
     * Breaks as the initiating player, including their current tool, enchantments,
     * protection events, drops, and experience.
     */
    PLAYER_TOOL,

    /**
     * Uses the platform's natural block-breaking operation without a player tool.
     */
    NATURAL_DROPS,

    /**
     * Breaks as the initiating player while suppressing item and experience drops.
     */
    NO_DROPS
}
