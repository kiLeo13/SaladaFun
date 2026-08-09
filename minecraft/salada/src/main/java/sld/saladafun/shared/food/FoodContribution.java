package sld.saladafun.shared.food;

import java.util.Objects;
import java.util.UUID;

/** One player's observed contribution to a shared-food tick. */
public record FoodContribution(
    UUID actorId,
    int foodLevelDelta,
    float saturationDelta,
    float exhaustionDelta
) {
    public FoodContribution {
        Objects.requireNonNull(actorId, "actorId");
        if (!Float.isFinite(saturationDelta) || !Float.isFinite(exhaustionDelta)) {
            throw new IllegalArgumentException("food deltas must be finite");
        }
    }
}
