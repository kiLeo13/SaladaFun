package sld.saladafun.shared.food;

import org.junit.jupiter.api.Test;

import java.util.List;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.assertEquals;

class SharedFoodTest {

    @Test
    void mergesSameTickLossAndGainAsDeltas() {
        SharedFood food = new SharedFood(new FoodState(10, 2.0F, 1.0F, 0));

        FoodState result = food.applyTick(List.of(
            contribution(-1, -1.0F, 0.5F),
            contribution(4, 3.0F, -0.25F)
        ));

        assertEquals(13, result.foodLevel());
        assertEquals(4.0F, result.saturation());
        assertEquals(1.25F, result.exhaustion());
        assertEquals(1, result.revision());
    }

    @Test
    void clampingAfterAggregationIsOrderIndependent() {
        FoodState initial = new FoodState(19, 19.0F, 39.0F, 3);
        List<FoodContribution> forward = List.of(
            contribution(4, 4.0F, 4.0F),
            contribution(-2, -2.0F, -2.0F)
        );
        List<FoodContribution> reverse = List.of(forward.get(1), forward.get(0));

        FoodState first = new SharedFood(initial).applyTick(forward);
        FoodState second = new SharedFood(initial).applyTick(reverse);

        assertEquals(first, second);
        assertEquals(20, first.foodLevel());
        assertEquals(20.0F, first.saturation());
        assertEquals(40.0F, first.exhaustion());
    }

    @Test
    void noEffectiveChangeDoesNotCreateARevision() {
        FoodState initial = FoodState.fresh();

        FoodState result = new SharedFood(initial).applyTick(List.of());

        assertEquals(initial, result);
    }

    private FoodContribution contribution(int food, float saturation, float exhaustion) {
        return new FoodContribution(UUID.randomUUID(), food, saturation, exhaustion);
    }
}
