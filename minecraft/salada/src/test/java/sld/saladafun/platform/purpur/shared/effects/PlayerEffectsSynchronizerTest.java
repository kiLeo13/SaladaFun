package sld.saladafun.platform.purpur.shared.effects;

import org.bukkit.Server;
import org.bukkit.entity.Player;
import org.junit.jupiter.api.Test;
import sld.saladafun.shared.effects.EffectState;
import sld.saladafun.shared.effects.EffectsState;

import java.util.List;
import java.util.Map;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

class PlayerEffectsSynchronizerTest {

    @Test
    void explicitChangesPropagateButNaturalAuditCountdownDoesNot() {
        UUID playerId = UUID.randomUUID();
        Player player = mock(Player.class);
        when(player.getUniqueId()).thenReturn(playerId);
        PurpurEffectsMapper mapper = mock(PurpurEffectsMapper.class);
        EffectState initialEffect = effect(200, 0);
        EffectState countdown = effect(180, 0);
        EffectState upgraded = effect(160, 1);
        EffectsState initial = state(initialEffect, 0);
        when(mapper.snapshot(player, 0)).thenReturn(
            initial,
            state(countdown, 0),
            state(upgraded, 0)
        );
        var synchronizer = new PlayerEffectsSynchronizer(
            mock(Server.class), mapper
        );
        synchronizer.apply(player, initial);

        var auditChanges = synchronizer.observe(player, List.of(), true);
        var explicitChanges = synchronizer.observe(
            player,
            List.of("minecraft:speed"),
            false
        );

        assertTrue(auditChanges.isEmpty());
        assertEquals(1, explicitChanges.size());
        assertEquals(upgraded, explicitChanges.getFirst().replacement().orElseThrow());
    }

    private EffectsState state(EffectState effect, long revision) {
        return new EffectsState(Map.of(effect.typeKey(), effect), revision);
    }

    private EffectState effect(int duration, int amplifier) {
        return new EffectState(
            "minecraft:speed", amplifier, duration, false, true, true
        );
    }
}
