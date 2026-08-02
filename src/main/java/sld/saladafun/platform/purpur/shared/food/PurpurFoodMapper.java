package sld.saladafun.platform.purpur.shared.food;

import org.bukkit.entity.Player;
import sld.saladafun.shared.food.FoodState;

import java.util.Objects;

/** Maps Purpur food mechanics into portable domain state. */
public final class PurpurFoodMapper {

    public FoodState snapshot(Player player, long revision) {
        Objects.requireNonNull(player, "player");
        return new FoodState(
            player.getFoodLevel(),
            player.getSaturation(),
            player.getExhaustion(),
            revision
        );
    }

    public void apply(Player player, FoodState state) {
        Objects.requireNonNull(player, "player");
        Objects.requireNonNull(state, "state");
        player.setFoodLevel(state.foodLevel());
        player.setSaturation(state.saturation());
        player.setExhaustion(state.exhaustion());
    }
}
