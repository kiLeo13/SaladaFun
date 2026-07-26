package sld.saladafun.shared.inventory.model;

import java.util.List;

/**
 * Stable, platform-independent identifiers for the inventory slots shared by the feature.
 */
public enum SlotKey {
    HOTBAR_0(true), HOTBAR_1(true), HOTBAR_2(true), HOTBAR_3(true), HOTBAR_4(true),
    HOTBAR_5(true), HOTBAR_6(true), HOTBAR_7(true), HOTBAR_8(true),
    MAIN_0(true), MAIN_1(true), MAIN_2(true), MAIN_3(true), MAIN_4(true), MAIN_5(true),
    MAIN_6(true), MAIN_7(true), MAIN_8(true), MAIN_9(true), MAIN_10(true), MAIN_11(true),
    MAIN_12(true), MAIN_13(true), MAIN_14(true), MAIN_15(true), MAIN_16(true), MAIN_17(true),
    MAIN_18(true), MAIN_19(true), MAIN_20(true), MAIN_21(true), MAIN_22(true), MAIN_23(true),
    MAIN_24(true), MAIN_25(true), MAIN_26(true),
    ARMOR_FEET(false), ARMOR_LEGS(false), ARMOR_CHEST(false), ARMOR_HEAD(false),
    OFF_HAND(false);

    private static final List<SlotKey> STORAGE_SLOTS = List.of(values()).stream()
        .filter(SlotKey::acceptsPickedUpItems)
        .toList();

    private final boolean acceptsPickedUpItems;

    SlotKey(boolean acceptsPickedUpItems) {
        this.acceptsPickedUpItems = acceptsPickedUpItems;
    }

    /**
     * Returns whether vanilla-style insertion may place a picked-up item in this slot.
     */
    public boolean acceptsPickedUpItems() {
        return acceptsPickedUpItems;
    }

    /**
     * Returns storage and hotbar slots in deterministic insertion order.
     */
    public static List<SlotKey> storageSlots() {
        return STORAGE_SLOTS;
    }
}
