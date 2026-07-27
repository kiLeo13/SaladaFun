package sld.saladafun.platform.purpur.discord;

import io.papermc.paper.event.player.AsyncChatEvent;
import net.kyori.adventure.text.serializer.plain.PlainTextComponentSerializer;
import org.bukkit.event.EventHandler;
import org.bukkit.event.EventPriority;
import org.bukkit.event.Listener;

import java.util.Objects;

/**
 * Observes the final accepted player chat without changing Minecraft delivery.
 */
public final class MinecraftChatListener implements Listener {
    private final MinecraftChatPublisher publisher;

    public MinecraftChatListener(MinecraftChatPublisher publisher) {
        this.publisher = Objects.requireNonNull(publisher, "publisher");
    }

    @EventHandler(priority = EventPriority.MONITOR, ignoreCancelled = true)
    public void onChat(AsyncChatEvent event) {
        publisher.publish(
            event.getPlayer().getName(),
            event.getPlayer().getUniqueId(),
            PlainTextComponentSerializer.plainText().serialize(event.message())
        );
    }
}
