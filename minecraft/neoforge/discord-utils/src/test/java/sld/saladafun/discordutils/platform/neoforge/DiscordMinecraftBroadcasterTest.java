package sld.saladafun.discordutils.platform.neoforge;

import static org.junit.jupiter.api.Assertions.assertEquals;

import java.util.Optional;
import net.minecraft.ChatFormatting;
import net.minecraft.network.chat.Component;
import org.junit.jupiter.api.Test;
import sld.saladafun.discordutils.discord.DiscordInboundMessage;
import sld.saladafun.discordutils.discord.DiscordReplyReference;

/** Tests Minecraft component rendering for Discord-originated messages. */
class DiscordMinecraftBroadcasterTest {
    @Test
    void ordinaryMessageRenderingRemainsUnchanged() {
        Component rendered = DiscordMinecraftBroadcaster.render(
            new DiscordInboundMessage("Leo13", "Oiee", 0, 0)
        );

        assertEquals("[Discord] Leo13: Oiee", rendered.getString());
    }

    @Test
    void replyReferenceIsRenderedAsOneSubduedLine() {
        DiscordReplyReference reference = new DiscordReplyReference(
            "  Lucas  ",
            "Hello,\n  guys",
            0,
            0
        );
        Component rendered = DiscordMinecraftBroadcaster.render(new DiscordInboundMessage(
            "Leo13",
            "Oiee, Lucas",
            0,
            0,
            Optional.of(reference)
        ));

        assertEquals(
            "┃ respondendo a Lucas: Hello, guys\n[Discord] Leo13: Oiee, Lucas",
            rendered.getString()
        );
        Component replyLine = rendered.getSiblings().getFirst();
        assertEquals(
            ChatFormatting.DARK_GRAY.getColor(),
            replyLine.getStyle().getColor().getValue()
        );
    }

    @Test
    void mediaOnlyReferenceUsesItsMediaSummary() {
        DiscordReplyReference reference = new DiscordReplyReference("Lucas", "", 2, 1);
        Component rendered = DiscordMinecraftBroadcaster.render(new DiscordInboundMessage(
            "Leo13",
            "Olha isso",
            0,
            0,
            Optional.of(reference)
        ));

        assertEquals(
            "┃ respondendo a Lucas: 2 image(s), 1 sticker(s)\n"
                + "[Discord] Leo13: Olha isso",
            rendered.getString()
        );
    }
}
