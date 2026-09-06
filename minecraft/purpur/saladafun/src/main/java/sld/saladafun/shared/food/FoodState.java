package sld.saladafun.shared.food;

/** Immutable canonical food, saturation, and exhaustion state. */
public record FoodState(
    int foodLevel,
    float saturation,
    float exhaustion,
    long revision
) {
    public static final int MAXIMUM_FOOD_LEVEL = 20;
    public static final float MAXIMUM_EXHAUSTION = 40.0F;

    public FoodState {
        if (foodLevel < 0 || foodLevel > MAXIMUM_FOOD_LEVEL) {
            throw new IllegalArgumentException("foodLevel must be between 0 and 20");
        }
        if (!Float.isFinite(saturation) || saturation < 0.0F || saturation > foodLevel) {
            throw new IllegalArgumentException("saturation must be between 0 and foodLevel");
        }
        if (!Float.isFinite(exhaustion)
            || exhaustion < 0.0F
            || exhaustion > MAXIMUM_EXHAUSTION) {
            throw new IllegalArgumentException("exhaustion must be between 0 and 40");
        }
        if (revision < 0) {
            throw new IllegalArgumentException("revision must not be negative");
        }
    }

    public static FoodState fresh() {
        return new FoodState(20, 5.0F, 0.0F, 0);
    }

    public FoodState withRevision(long newRevision) {
        return new FoodState(foodLevel, saturation, exhaustion, newRevision);
    }
}
