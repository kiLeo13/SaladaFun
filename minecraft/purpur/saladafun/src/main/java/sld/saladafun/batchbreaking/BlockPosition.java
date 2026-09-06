package sld.saladafun.batchbreaking;

/**
 * Platform-independent integer world position.
 */
public record BlockPosition(int x, int y, int z) {
    public int chunkX() {
        return Math.floorDiv(x, 16);
    }

    public int chunkZ() {
        return Math.floorDiv(z, 16);
    }
}
