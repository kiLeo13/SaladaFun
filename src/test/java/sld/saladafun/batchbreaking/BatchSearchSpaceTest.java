package sld.saladafun.batchbreaking;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

class BatchSearchSpaceTest {

    @Test
    void radiusOneScansOnlyTheTwentySevenCoordinateCube() {
        var space = new BatchSearchSpace(
            new BlockPosition(8, 64, 8),
            new BatchBreakingSetting.CubicRadius(1),
            -64,
            320
        );

        assertTrue(space.intersectsChunk(0, 0));
        assertFalse(space.intersectsChunk(1, 0));
        assertEquals(7, space.minimumLocalX(0));
        assertEquals(10, space.maximumLocalXExclusive(0));
        assertEquals(63, space.minimumY());
        assertEquals(66, space.maximumYExclusive());
        assertEquals(7, space.minimumLocalZ(0));
        assertEquals(10, space.maximumLocalZExclusive(0));
    }

    @Test
    void finiteRangeIncludesAdjacentChunksAtAChunkBoundary() {
        var space = new BatchSearchSpace(
            new BlockPosition(15, 64, 15),
            new BatchBreakingSetting.CubicRadius(1),
            -64,
            320
        );

        assertTrue(space.intersectsChunk(0, 0));
        assertTrue(space.intersectsChunk(1, 0));
        assertTrue(space.intersectsChunk(0, 1));
        assertTrue(space.intersectsChunk(1, 1));
        assertFalse(space.intersectsChunk(-1, 0));
        assertEquals(14, space.minimumLocalX(0));
        assertEquals(16, space.maximumLocalXExclusive(0));
        assertEquals(0, space.minimumLocalX(1));
        assertEquals(1, space.maximumLocalXExclusive(1));
    }

    @Test
    void allUsesFullSentChunksAndWorldHeight() {
        var space = new BatchSearchSpace(
            new BlockPosition(0, 64, 0),
            new BatchBreakingSetting.AllSentChunks(),
            -64,
            320
        );

        assertTrue(space.intersectsChunk(-100, 100));
        assertEquals(0, space.minimumLocalX(-100));
        assertEquals(16, space.maximumLocalXExclusive(100));
        assertEquals(-64, space.minimumY());
        assertEquals(320, space.maximumYExclusive());
    }
}
