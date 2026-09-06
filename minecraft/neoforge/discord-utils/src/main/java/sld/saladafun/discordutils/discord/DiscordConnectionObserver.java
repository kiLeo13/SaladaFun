package sld.saladafun.discordutils.discord;

/** Receives a candidate Discord session's terminal connection outcomes. */
interface DiscordConnectionObserver {
    /** Marks session ready after channel visibility validation succeeds. */
    void ready(DiscordSession session);

    /** Rejects session because startup or validation failed. */
    void failed(DiscordSession session, Throwable failure);
}
