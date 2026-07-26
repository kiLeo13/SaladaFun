package sld.saladafun.batchbreaking;

import java.util.Objects;

/**
 * Precomputed, overflow-safe scan bounds for a batch-breaking trigger.
 */
public final class BatchSearchSpace {
    private static final int CHUNK_SIZE = 16;

    private final CubicRange range = new CubicRange();
    private final BlockPosition origin;
    private final BatchBreakingSetting setting;
    private final long minimumX;
    private final long maximumX;
    private final int minimumY;
    private final int maximumYExclusive;
    private final long minimumZ;
    private final long maximumZ;

    public BatchSearchSpace(
        BlockPosition origin,
        BatchBreakingSetting setting,
        int worldMinimumY,
        int worldMaximumY
    ) {
        this.origin = Objects.requireNonNull(origin, "origin");
        this.setting = Objects.requireNonNull(setting, "setting");
        if (worldMinimumY >= worldMaximumY) {
            throw new IllegalArgumentException("World height range must not be empty");
        }

        if (setting instanceof BatchBreakingSetting.CubicRadius radius) {
            minimumX = (long) origin.x() - radius.blocks();
            maximumX = (long) origin.x() + radius.blocks();
            minimumY = (int) Math.max(
                worldMinimumY, (long) origin.y() - radius.blocks()
            );
            maximumYExclusive = (int) Math.min(
                worldMaximumY, (long) origin.y() + radius.blocks() + 1L
            );
            minimumZ = (long) origin.z() - radius.blocks();
            maximumZ = (long) origin.z() + radius.blocks();
        } else {
            minimumX = Integer.MIN_VALUE;
            maximumX = Integer.MAX_VALUE;
            minimumY = worldMinimumY;
            maximumYExclusive = worldMaximumY;
            minimumZ = Integer.MIN_VALUE;
            maximumZ = Integer.MAX_VALUE;
        }
    }

    public boolean intersectsChunk(int chunkX, int chunkZ) {
        if (setting instanceof BatchBreakingSetting.Disabled) {
            return false;
        }
        if (setting instanceof BatchBreakingSetting.AllSentChunks) {
            return true;
        }
        long chunkMinimumX = (long) chunkX * CHUNK_SIZE;
        long chunkMaximumX = chunkMinimumX + CHUNK_SIZE - 1L;
        long chunkMinimumZ = (long) chunkZ * CHUNK_SIZE;
        long chunkMaximumZ = chunkMinimumZ + CHUNK_SIZE - 1L;
        return chunkMaximumX >= minimumX
            && chunkMinimumX <= maximumX
            && chunkMaximumZ >= minimumZ
            && chunkMinimumZ <= maximumZ;
    }

    public int minimumLocalX(int chunkX) {
        return minimumLocalCoordinate(chunkX, minimumX);
    }

    public int maximumLocalXExclusive(int chunkX) {
        return maximumLocalCoordinateExclusive(chunkX, maximumX);
    }

    public int minimumLocalZ(int chunkZ) {
        return minimumLocalCoordinate(chunkZ, minimumZ);
    }

    public int maximumLocalZExclusive(int chunkZ) {
        return maximumLocalCoordinateExclusive(chunkZ, maximumZ);
    }

    public int minimumY() {
        return minimumY;
    }

    public int maximumYExclusive() {
        return maximumYExclusive;
    }

    public boolean contains(BlockPosition position) {
        Objects.requireNonNull(position, "position");
        return position.y() >= minimumY
            && position.y() < maximumYExclusive
            && range.contains(origin, position, setting);
    }

    private int minimumLocalCoordinate(int chunkCoordinate, long minimum) {
        long chunkMinimum = (long) chunkCoordinate * CHUNK_SIZE;
        return (int) Math.max(0L, minimum - chunkMinimum);
    }

    private int maximumLocalCoordinateExclusive(int chunkCoordinate, long maximum) {
        long chunkMinimum = (long) chunkCoordinate * CHUNK_SIZE;
        return (int) Math.min(CHUNK_SIZE, maximum - chunkMinimum + 1L);
    }
}
