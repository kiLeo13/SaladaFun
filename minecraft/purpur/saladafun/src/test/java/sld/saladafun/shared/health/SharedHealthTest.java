package sld.saladafun.shared.health;

import org.junit.jupiter.api.Test;

import java.util.List;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.assertEquals;

class SharedHealthTest {

    @Test
    void mergesDamageHealingAndAbsorptionInOneRevision() {
        SharedHealth health = new SharedHealth(
            new HealthState(10.0, 20.0, 2.0, 10.0, HealthPhase.ALIVE, 0)
        );

        HealthState result = health.applyTick(List.of(
            contribution(-1.0, -2.0),
            contribution(4.0, 3.0)
        ), false);

        assertEquals(13.0, result.health());
        assertEquals(3.0, result.absorption());
        assertEquals(1, result.revision());
    }

    @Test
    void lethalTickOverridesHealingAndClearsAbsorption() {
        SharedHealth health = new SharedHealth(
            new HealthState(2.0, 20.0, 4.0, 10.0, HealthPhase.ALIVE, 8)
        );

        HealthState result = health.applyTick(
            List.of(contribution(10.0, 3.0)),
            true
        );

        assertEquals(0.0, result.health());
        assertEquals(0.0, result.absorption());
        assertEquals(HealthPhase.DEAD, result.phase());
        assertEquals(9, result.revision());
    }

    @Test
    void rangeWriteIsAbsoluteAndClampsCurrentValues() {
        UUID actor = new UUID(0, 2);
        SharedHealth health = new SharedHealth(
            new HealthState(20.0, 20.0, 5.0, 10.0, HealthPhase.ALIVE, 0)
        );

        HealthState result = health.applyTick(List.of(
            new HealthContribution(actor, -16.0, 0.0, true, 4.0, 2.0)
        ), false);

        assertEquals(4.0, result.maximumHealth());
        assertEquals(4.0, result.health());
        assertEquals(2.0, result.maximumAbsorption());
        assertEquals(2.0, result.absorption());
    }

    @Test
    void sameTickRangeConflictHasDeterministicLastWriter() {
        HealthContribution lower = new HealthContribution(
            new UUID(0, 1), 0.0, 0.0, true, 4.0, 2.0
        );
        HealthContribution higher = new HealthContribution(
            new UUID(0, 2), 0.0, 0.0, true, 30.0, 12.0
        );
        HealthState initial = HealthState.full(20.0, 10.0);

        HealthState forward = new SharedHealth(initial).applyTick(
            List.of(lower, higher), false
        );
        HealthState reverse = new SharedHealth(initial).applyTick(
            List.of(higher, lower), false
        );

        assertEquals(forward, reverse);
        assertEquals(30.0, forward.maximumHealth());
        assertEquals(12.0, forward.maximumAbsorption());
    }

    @Test
    void respawnRevivesAtFullCanonicalRange() {
        SharedHealth health = new SharedHealth(
            new HealthState(0.0, 4.0, 0.0, 2.0, HealthPhase.DEAD, 2)
        );

        HealthState result = health.revive();

        assertEquals(4.0, result.health());
        assertEquals(HealthPhase.ALIVE, result.phase());
        assertEquals(3, result.revision());
    }

    private HealthContribution contribution(double health, double absorption) {
        return new HealthContribution(
            UUID.randomUUID(), health, absorption, false, 20.0, 10.0
        );
    }
}
