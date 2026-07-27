package sld.saladafun.platform.purpur.discord;

import net.kyori.adventure.text.Component;
import net.kyori.adventure.text.serializer.legacy.LegacyComponentSerializer;
import net.kyori.adventure.text.serializer.plain.PlainTextComponentSerializer;
import org.bukkit.Server;
import org.bukkit.plugin.java.JavaPlugin;
import org.bukkit.scheduler.BukkitScheduler;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class DiscordMinecraftBroadcasterTest {

    @Test
    void rendersMediaCountersOnTheNextLineWithCorrectPluralsAndColor() {
        Component rendered = DiscordMinecraftBroadcaster.render(
            new DiscordInboundMessage("Alex", "hello", 2, 1)
        );

        assertEquals(
            "[Discord] <Alex> hello\n[+2 images] [+1 sticker]",
            PlainTextComponentSerializer.plainText().serialize(rendered)
        );
        String legacy = LegacyComponentSerializer.legacySection().serialize(rendered);
        assertTrue(legacy.contains("§d[+2 images]"));
        assertTrue(legacy.contains("§d[+1 sticker]"));
    }

    @Test
    void rendersMediaOnlyMessagesWithoutInventingText() {
        Component rendered = DiscordMinecraftBroadcaster.render(
            new DiscordInboundMessage("Steve", "", 1, 2)
        );

        assertEquals(
            "[Discord] <Steve>\n[+1 image] [+2 stickers]",
            PlainTextComponentSerializer.plainText().serialize(rendered)
        );
    }

    @Test
    void schedulesDiscordBroadcastsOnTheMinecraftThread() {
        JavaPlugin plugin = mock(JavaPlugin.class);
        Server server = mock(Server.class);
        BukkitScheduler scheduler = mock(BukkitScheduler.class);
        when(plugin.getServer()).thenReturn(server);
        when(plugin.isEnabled()).thenReturn(true);
        when(server.getScheduler()).thenReturn(scheduler);
        Runnable[] scheduled = new Runnable[1];
        when(scheduler.runTask(any(JavaPlugin.class), any(Runnable.class)))
            .thenAnswer(invocation -> {
                scheduled[0] = invocation.getArgument(1);
                return null;
            });

        new DiscordMinecraftBroadcaster(plugin).accept(
            new DiscordInboundMessage("Alex", "hello", 0, 0)
        );

        verify(server, never()).broadcast(any(Component.class));
        scheduled[0].run();
        verify(server).broadcast(any(Component.class));
    }

    @Test
    void dropsLateDiscordCallbacksAfterPluginShutdown() {
        JavaPlugin plugin = mock(JavaPlugin.class);
        Server server = mock(Server.class);
        when(plugin.getServer()).thenReturn(server);
        when(plugin.isEnabled()).thenReturn(false);

        new DiscordMinecraftBroadcaster(plugin).accept(
            new DiscordInboundMessage("Alex", "late", 0, 0)
        );

        verify(server, never()).getScheduler();
    }
}
