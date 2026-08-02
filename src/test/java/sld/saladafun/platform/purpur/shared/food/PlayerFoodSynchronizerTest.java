package sld.saladafun.platform.purpur.shared.food;

import org.bukkit.Server;
import org.bukkit.entity.Player;
import org.junit.jupiter.api.Test;
import sld.saladafun.shared.food.FoodState;
import sld.saladafun.shared.food.SharedFood;

import java.util.Set;
import java.util.UUID;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicReference;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.mockito.Mockito.doAnswer;
import static org.mockito.Mockito.doReturn;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

class PlayerFoodSynchronizerTest {

    @Test
    void observesEveryPlayersNetDeltaWithoutCountingSynchronizationWrites() {
        MutableFood firstState = new MutableFood(20, 5.0F, 0.0F);
        MutableFood secondState = new MutableFood(20, 5.0F, 0.0F);
        Player first = player(UUID.randomUUID(), firstState);
        Player second = player(UUID.randomUUID(), secondState);
        Server server = mock(Server.class);
        doReturn(Set.of(first, second)).when(server).getOnlinePlayers();
        PlayerFoodSynchronizer synchronizer = new PlayerFoodSynchronizer(
            server, new PurpurFoodMapper()
        );
        FoodState canonical = new FoodState(10, 2.0F, 1.0F, 0);
        synchronizer.applyToAll(canonical);

        firstState.set(9, 1.0F, 1.5F);
        secondState.set(14, 5.0F, 0.75F);
        FoodState result = new SharedFood(canonical).applyTick(
            synchronizer.observeOnline()
        );

        assertEquals(13, result.foodLevel());
        assertEquals(4.0F, result.saturation());
        assertEquals(1.25F, result.exhaustion());
    }

    private Player player(UUID id, MutableFood state) {
        Player player = mock(Player.class);
        when(player.getUniqueId()).thenReturn(id);
        when(player.getFoodLevel()).thenAnswer(invocation -> state.food.get());
        when(player.getSaturation()).thenAnswer(invocation -> state.saturation.get());
        when(player.getExhaustion()).thenAnswer(invocation -> state.exhaustion.get());
        doAnswer(invocation -> {
            state.food.set(invocation.getArgument(0));
            return null;
        }).when(player).setFoodLevel(org.mockito.ArgumentMatchers.anyInt());
        doAnswer(invocation -> {
            state.saturation.set(invocation.getArgument(0));
            return null;
        }).when(player).setSaturation(org.mockito.ArgumentMatchers.anyFloat());
        doAnswer(invocation -> {
            state.exhaustion.set(invocation.getArgument(0));
            return null;
        }).when(player).setExhaustion(org.mockito.ArgumentMatchers.anyFloat());
        return player;
    }

    private static final class MutableFood {
        private final AtomicInteger food = new AtomicInteger();
        private final AtomicReference<Float> saturation = new AtomicReference<>();
        private final AtomicReference<Float> exhaustion = new AtomicReference<>();

        private MutableFood(int food, float saturation, float exhaustion) {
            set(food, saturation, exhaustion);
        }

        private void set(int food, float saturation, float exhaustion) {
            this.food.set(food);
            this.saturation.set(saturation);
            this.exhaustion.set(exhaustion);
        }
    }
}
