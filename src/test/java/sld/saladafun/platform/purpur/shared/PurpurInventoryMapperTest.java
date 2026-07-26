package sld.saladafun.platform.purpur.shared;

import org.junit.jupiter.api.Test;
import sld.saladafun.shared.inventory.model.SlotKey;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

class PurpurInventoryMapperTest {
    private final PurpurInventoryMapper mapper = new PurpurInventoryMapper();

    @Test
    void mapsEverySharedSlotWithoutIncludingHeldSelection() {
        for (int slot = 0; slot <= 40; slot++) {
            SlotKey core = mapper.fromBukkitSlot(slot).orElseThrow();
            assertEquals(slot, mapper.toBukkitSlot(core));
        }
        assertTrue(mapper.fromBukkitSlot(41).isEmpty());
    }
}
