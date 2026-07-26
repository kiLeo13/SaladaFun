package sld.saladafun.shared.inventory.model;

/**
 * Outcome of an authoritative inventory mutation.
 */
public enum MutationStatus {
    ACCEPTED,
    STALE_STATE,
    ITEM_UNAVAILABLE,
    INVENTORY_FULL,
    SLOT_RESERVED,
    UNKNOWN_OPERATION,
    ALREADY_COMPLETED
}
