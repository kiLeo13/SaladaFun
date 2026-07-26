package sld.saladafun.batchbreaking;

import java.util.Objects;

/**
 * Overflow-safe inclusive Chebyshev range used by positive batch-breaking settings.
 */
public final class CubicRange {
    public boolean contains(
        BlockPosition origin,
        BlockPosition candidate,
        BatchBreakingSetting setting
    ) {
        Objects.requireNonNull(origin, "origin");
        Objects.requireNonNull(candidate, "candidate");
        Objects.requireNonNull(setting, "setting");
        return switch (setting) {
            case BatchBreakingSetting.Disabled ignored -> false;
            case BatchBreakingSetting.AllSentChunks ignored -> true;
            case BatchBreakingSetting.CubicRadius radius -> {
                long dx = Math.abs((long) candidate.x() - origin.x());
                long dy = Math.abs((long) candidate.y() - origin.y());
                long dz = Math.abs((long) candidate.z() - origin.z());
                yield Math.max(dx, Math.max(dy, dz)) <= radius.blocks();
            }
        };
    }
}
