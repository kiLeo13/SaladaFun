package sld.saladafun.discordutils.platform.neoforge;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertThrows;

import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.atomic.AtomicReference;
import org.junit.jupiter.api.Test;

/** Verifies player names are copied only after work reaches the Minecraft server thread. */
class MinecraftOnlinePlayerNamesProviderTest {
    @Test
    void schedulesAndDefensivelyCopiesOnlinePlayerSnapshot() {
        List<Runnable> serverTasks = new ArrayList<>();
        List<String> onlinePlayers = new ArrayList<>(List.of("First"));
        AtomicReference<List<String>> received = new AtomicReference<>();
        MinecraftOnlinePlayerNamesProvider provider = new MinecraftOnlinePlayerNamesProvider(
            serverTasks::add,
            () -> onlinePlayers
        );

        provider.request(received::set);
        assertNull(received.get());
        onlinePlayers.add("Second");
        serverTasks.getFirst().run();
        onlinePlayers.add("Third");

        assertEquals(List.of("First", "Second"), received.get());
        assertThrows(UnsupportedOperationException.class, () -> received.get().add("Fourth"));
    }
}
