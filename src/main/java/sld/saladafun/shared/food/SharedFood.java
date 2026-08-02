package sld.saladafun.shared.food;

import java.util.List;
import java.util.Objects;

/** Thread-safe canonical food aggregate with tick-batched delta merging. */
public final class SharedFood {
    private FoodState state;

    public SharedFood(FoodState initialState) {
        state = Objects.requireNonNull(initialState, "initialState");
    }

    public synchronized FoodState snapshot() {
        return state;
    }

    /** Adds all deltas and clamps each canonical component exactly once. */
    public synchronized FoodState applyTick(List<FoodContribution> contributions) {
        Objects.requireNonNull(contributions, "contributions");
        int foodDelta = contributions.stream()
            .mapToInt(FoodContribution::foodLevelDelta)
            .sum();
        double saturationDelta = contributions.stream()
            .mapToDouble(FoodContribution::saturationDelta)
            .sum();
        double exhaustionDelta = contributions.stream()
            .mapToDouble(FoodContribution::exhaustionDelta)
            .sum();

        int foodLevel = clamp(
            state.foodLevel() + foodDelta,
            0,
            FoodState.MAXIMUM_FOOD_LEVEL
        );
        float saturation = (float) clamp(
            state.saturation() + saturationDelta,
            0.0,
            foodLevel
        );
        float exhaustion = (float) clamp(
            state.exhaustion() + exhaustionDelta,
            0.0,
            FoodState.MAXIMUM_EXHAUSTION
        );
        FoodState candidate = new FoodState(
            foodLevel,
            saturation,
            exhaustion,
            state.revision()
        );
        if (!sameValues(state, candidate)) {
            state = candidate.withRevision(state.revision() + 1);
        }
        return state;
    }

    private boolean sameValues(FoodState left, FoodState right) {
        return left.foodLevel() == right.foodLevel()
            && Float.compare(left.saturation(), right.saturation()) == 0
            && Float.compare(left.exhaustion(), right.exhaustion()) == 0;
    }

    private int clamp(int value, int minimum, int maximum) {
        return Math.max(minimum, Math.min(value, maximum));
    }

    private double clamp(double value, double minimum, double maximum) {
        return Math.max(minimum, Math.min(value, maximum));
    }
}
