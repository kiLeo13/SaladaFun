package sld.saladafun.shared.effects;

import org.junit.jupiter.api.Test;

import java.util.List;
import java.util.Map;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;

class SharedEffectsTest {

    @Test
    void mergesDifferentTypesAndUsesDeterministicLwwPerType() {
        UUID lower = UUID.fromString("00000000-0000-0000-0000-000000000001");
        UUID higher = UUID.fromString("ffffffff-ffff-ffff-ffff-ffffffffffff");
        EffectState speedOne = effect("minecraft:speed", 0, 200);
        EffectState speedTwo = effect("minecraft:speed", 1, 100);
        EffectState strength = effect("minecraft:strength", 0, 300);
        SharedEffects aggregate = new SharedEffects(EffectsState.empty());

        EffectsState result = aggregate.applyTick(List.of(
            EffectChange.replace(lower, speedOne),
            EffectChange.replace(higher, speedTwo),
            EffectChange.replace(lower, strength)
        ));

        assertEquals(speedTwo, result.effects().get("minecraft:speed"));
        assertEquals(strength, result.effects().get("minecraft:strength"));
        assertEquals(1, result.revision());
    }

    @Test
    void propagatesRemovalAndRefreshesDurationWithoutGameplayRevision() {
        EffectState speed = effect("minecraft:speed", 0, 200);
        SharedEffects aggregate = new SharedEffects(new EffectsState(
            Map.of(speed.typeKey(), speed), 4
        ));

        EffectsState refreshed = aggregate.refreshDurations(Map.of(
            speed.typeKey(), effect("minecraft:speed", 0, 180)
        ));
        EffectsState removed = aggregate.applyTick(List.of(
            EffectChange.remove(UUID.randomUUID(), speed.typeKey())
        ));

        assertEquals(180, refreshed.effects().get(speed.typeKey()).durationTicks());
        assertEquals(4, refreshed.revision());
        assertFalse(removed.effects().containsKey(speed.typeKey()));
        assertEquals(5, removed.revision());
    }

    private EffectState effect(String type, int amplifier, int duration) {
        return new EffectState(type, amplifier, duration, false, true, true);
    }
}
