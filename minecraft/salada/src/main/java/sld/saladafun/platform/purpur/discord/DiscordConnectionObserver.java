package sld.saladafun.platform.purpur.discord;

interface DiscordConnectionObserver {
    void ready(DiscordSession session);

    void failed(DiscordSession session, Throwable failure);
}
