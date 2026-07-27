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

import java.time.Duration;
import java.util.Objects;
import java.util.UUID;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.function.Consumer;
import java.util.logging.Logger;

final class JdaDiscordSession implements DiscordSession {
    private static final Duration ABANDONED_SESSION_TERMINATION_TIMEOUT =
        Duration.ofSeconds(10);

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
            .setEnableShutdownHook(false)
            .build();
        try {
            IncomingWebhookClient webhook = WebhookClient.createClient(
                candidate,
                settings.webhookUrl()
            );
            this.webhookSender = new DiscordWebhookSender(webhook, logger);
            this.jda = candidate;
        } catch (RuntimeException exception) {
            closeAbandonedCandidate(candidate, logger);
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
    public boolean awaitTermination(Duration timeout) throws InterruptedException {
        return jda.awaitShutdown(timeout);
    }

    @Override
    public void close() {
        active.set(false);
        if (closed.compareAndSet(false, true)) {
            jda.shutdownNow();
        }
    }

    private static void closeAbandonedCandidate(JDA candidate, Logger logger) {
        candidate.shutdownNow();
        try {
            if (!candidate.awaitShutdown(ABANDONED_SESSION_TERMINATION_TIMEOUT)) {
                logger.warning(
                    "Abandoned Discord session did not stop within "
                        + ABANDONED_SESSION_TERMINATION_TIMEOUT.toSeconds()
                        + " seconds."
                );
            }
        } catch (InterruptedException exception) {
            Thread.currentThread().interrupt();
            logger.warning(
                "Interrupted while waiting for an abandoned Discord session to stop."
            );
        }
    }
}
