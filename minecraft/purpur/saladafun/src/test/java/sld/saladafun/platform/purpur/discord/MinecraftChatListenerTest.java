package sld.saladafun.platform.purpur.discord;

import io.papermc.paper.event.player.AsyncChatEvent;
import net.kyori.adventure.text.Component;
import org.bukkit.entity.Player;
import org.junit.jupiter.api.Test;

import java.util.UUID;

import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class MinecraftChatListenerTest {

    @Test
    void publishesTheFinalAdventureMessageAsPlainText() {
        MinecraftChatPublisher publisher = mock(MinecraftChatPublisher.class);
        AsyncChatEvent event = mock(AsyncChatEvent.class);
        Player player = mock(Player.class);
        UUID playerId = UUID.randomUUID();
        when(event.getPlayer()).thenReturn(player);
        when(event.message()).thenReturn(
            Component.text("hello ").append(Component.text("world"))
        );
        when(player.getName()).thenReturn("Alex");
        when(player.getUniqueId()).thenReturn(playerId);

        new MinecraftChatListener(publisher).onChat(event);

        verify(publisher).publish("Alex", playerId, "hello world");
    }
}
