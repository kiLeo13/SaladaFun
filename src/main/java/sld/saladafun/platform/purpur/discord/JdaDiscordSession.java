package sld.saladafun.platform.purpur.discord;

import net.dv8tion.jda.api.JDA;
import net.dv8tion.jda.api.JDABuilder;
import net.dv8tion.jda.api.entities.IncomingWebhookClient;
import net.dv8tion.jda.api.entities.WebhookClient;
import net.dv8tion.jda.api.entities.channel.middleman.GuildMessageChannel;
import net.dv8tion.jda.api.events.session.ReadyEvent;
import net.dv8tion.jda.api.events.session.ShutdownEvent;
import net.dv8tion.jda.api.hooks.ListenerAdapter;
import net.dv8tion.jda.api.requests.GatewayIntent;
import sld.saladafun.platform.purpur.config.DiscordChatSettings;

import java.util.Objects;
import java.util.UUID;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.function.Consumer;
import java.util.logging.Logger;

final class JdaDiscordSession implements DiscordSession {
    private final DiscordChatSettings settings;
    private final AtomicBoolean active = new AtomicBoolean();
    private final AtomicBoolean ready = new AtomicBoolean();
    private final AtomicBoolean closed = new AtomicBoolean();
    private final JDA jda;
    private final DiscordWebhookSender webhookSender;

    JdaDiscordSession(
        DiscordChatSettings settings,
        Consumer<DiscordInboundMessage> destination,
        DiscordConnectionObserver observer,
        Logger logger
    ) {
        this.settings = Objects.requireNonNull(settings, "settings");
        Objects.requireNonNull(logger, "logger");
        Objects.requireNonNull(destination, "destination");
        Objects.requireNonNull(observer, "observer");

        var messages = new DiscordMessageListener(
            settings.channelId(),
            active::get,
            destination
        );
        var lifecycle = new ListenerAdapter() {
            @Override
            public void onReady(ReadyEvent event) {
                if (closed.get()) {
                    return;
                }
                if (!(event.getJDA().getGuildChannelById(settings.channelId())
                    instanceof GuildMessageChannel)) {
                    observer.failed(
                        JdaDiscordSession.this,
                        new IllegalStateException("Configured Discord text channel is unavailable")
                    );
                    return;
                }
                ready.set(true);
                observer.ready(JdaDiscordSession.this);
            }

            @Override
            public void onShutdown(ShutdownEvent event) {
                if (!closed.get() && !ready.get()) {
                    observer.failed(
                        JdaDiscordSession.this,
                        new IllegalStateException(
                            "Discord gateway closed before becoming ready (code "
                                + event.getCode() + ")"
                        )
                    );
                }
            }
        };

        JDA candidate = JDABuilder.createLight(
                settings.token(),
                GatewayIntent.GUILD_MESSAGES,
                GatewayIntent.MESSAGE_CONTENT
            )
            .addEventListeners(messages, lifecycle)
            .build();
        try {
            IncomingWebhookClient webhook = WebhookClient.createClient(
                candidate,
                settings.webhookUrl()
            );
            this.webhookSender = new DiscordWebhookSender(webhook, logger);
            this.jda = candidate;
        } catch (RuntimeException exception) {
            candidate.shutdownNow();
            throw exception;
        }
    }

    @Override
    public DiscordChatSettings settings() {
        return settings;
    }

    @Override
    public void activate() {
        if (!closed.get()) {
            active.set(true);
        }
    }

    @Override
    public void publish(String playerName, UUID playerId, String content) {
        if (!active.get() || content.isBlank()) {
            return;
        }
        webhookSender.send(playerName, playerId, content);
    }

    @Override
    public void close() {
        active.set(false);
        if (closed.compareAndSet(false, true)) {
            jda.shutdownNow();
        }
    }
}
