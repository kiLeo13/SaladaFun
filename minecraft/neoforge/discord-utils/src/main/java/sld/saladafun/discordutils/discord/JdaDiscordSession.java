package sld.saladafun.discordutils.discord;

import java.time.Duration;
import java.util.Objects;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.function.Consumer;
import net.dv8tion.jda.api.JDA;
import net.dv8tion.jda.api.JDABuilder;
import net.dv8tion.jda.api.entities.IncomingWebhookClient;
import net.dv8tion.jda.api.entities.WebhookClient;
import net.dv8tion.jda.api.entities.channel.middleman.GuildMessageChannel;
import net.dv8tion.jda.api.events.session.ReadyEvent;
import net.dv8tion.jda.api.events.session.ShutdownEvent;
import net.dv8tion.jda.api.hooks.ListenerAdapter;
import net.dv8tion.jda.api.requests.GatewayIntent;
import org.slf4j.Logger;
import sld.saladafun.discordutils.config.DiscordChatSettings;

/** JDA implementation of one candidate or active Discord transport session. */
final class JdaDiscordSession implements DiscordSession {
    private final DiscordChatSettings settings;
    private final AtomicBoolean active = new AtomicBoolean();
    private final AtomicBoolean ready = new AtomicBoolean();
    private final AtomicBoolean closed = new AtomicBoolean();
    private final JDA jda;
    private final DiscordWebhookSender webhookSender;
    private final Logger logger;

    private volatile DiscordPlayerNotificationSender playerNotificationSender;

    /** Connects a lightweight JDA session without activating message traffic yet. */
    JdaDiscordSession(
        DiscordChatSettings settings,
        Consumer<DiscordInboundMessage> destination,
        DiscordConnectionObserver observer,
        Logger logger
    ) {
        this.settings = Objects.requireNonNull(settings, "settings");
        Objects.requireNonNull(destination, "destination");
        Objects.requireNonNull(observer, "observer");
        this.logger = Objects.requireNonNull(logger, "logger");

        DiscordMessageListener messages = new DiscordMessageListener(
            settings.channelId(),
            active::get,
            destination
        );
        ListenerAdapter lifecycle = lifecycleListener(observer);
        JDA candidate = JDABuilder.createLight(
            settings.token(),
            GatewayIntent.GUILD_MESSAGES,
            GatewayIntent.MESSAGE_CONTENT
        )
            .addEventListeners(messages, lifecycle)
            .setEnableShutdownHook(false)
            .build();
        this.jda = candidate;

        try {
            IncomingWebhookClient webhook = WebhookClient.createClient(candidate, settings.webhookUrl());
            this.webhookSender = new DiscordWebhookSender(webhook, logger);
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
    public void publish(String playerName, String content) {
        if (active.get()) {
            webhookSender.send(playerName, content);
        }
    }

    @Override
    public void publishPresence(String playerName, PlayerPresence presence) {
        DiscordPlayerNotificationSender sender = playerNotificationSender;
        if (active.get() && sender != null) {
            sender.send(playerName, presence);
        }
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

    private ListenerAdapter lifecycleListener(DiscordConnectionObserver observer) {
        return new ListenerAdapter() {
            @Override
            public void onReady(ReadyEvent event) {
                if (closed.get()) {
                    return;
                }

                if (!(event.getJDA().getGuildChannelById(settings.channelId())
                    instanceof GuildMessageChannel channel)) {
                    observer.failed(
                        JdaDiscordSession.this,
                        new IllegalStateException("Configured Discord text channel is unavailable")
                    );
                    return;
                }

                playerNotificationSender = new DiscordPlayerNotificationSender(channel, logger);
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
                                + event.getCode()
                                + ")"
                        )
                    );
                }
            }
        };
    }
}
