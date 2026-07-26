package sld.saladafun.batchbreaking;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

class BatchBreakingSettingTest {
    private final BatchBreakingSettingParser parser = new BatchBreakingSettingParser();
    private final CubicRange range = new CubicRange();

    @Test
    void parsesHumanReadableSettings() {
        assertEquals(new BatchBreakingSetting.Disabled(), parser.parse("disabled"));
        assertEquals(new BatchBreakingSetting.AllSentChunks(), parser.parse("ALL"));
        assertEquals(new BatchBreakingSetting.CubicRadius(12), parser.parse("12"));
        assertEquals("12", parser.format(parser.parse("12")));
    }

    @Test
    void rejectsZeroNegativeAndOverflow() {
        assertThrows(IllegalArgumentException.class, () -> parser.parse("0"));
        assertThrows(IllegalArgumentException.class, () -> parser.parse("-1"));
        assertThrows(IllegalArgumentException.class, () -> parser.parse("2147483648"));
        assertThrows(IllegalArgumentException.class, () -> parser.parse("many"));
    }

    @Test
    void radiusOneIncludesAllTwentySixNeighbors() {
        BlockPosition origin = new BlockPosition(0, 64, 0);
        var setting = new BatchBreakingSetting.CubicRadius(1);
        int included = 0;

        for (int x = -1; x <= 1; x++) {
            for (int y = 63; y <= 65; y++) {
                for (int z = -1; z <= 1; z++) {
                    BlockPosition candidate = new BlockPosition(x, y, z);
                    if (!candidate.equals(origin) && range.contains(origin, candidate, setting)) {
                        included++;
                    }
                }
            }
        }

        assertEquals(26, included);
        assertFalse(range.contains(origin, new BlockPosition(2, 64, 0), setting));
    }

    @Test
    void hugeCoordinatesDoNotOverflow() {
        var maximum = new BatchBreakingSetting.CubicRadius(Integer.MAX_VALUE);
        assertTrue(range.contains(
            new BlockPosition(Integer.MIN_VALUE, 0, 0),
            new BlockPosition(-1, 0, 0),
            maximum
        ));
        assertFalse(range.contains(
            new BlockPosition(Integer.MIN_VALUE, 0, 0),
            new BlockPosition(Integer.MAX_VALUE, 0, 0),
            maximum
        ));
    }
}
