-- +goose Up
CREATE TABLE voice_activity_logs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    guild_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    old_channel_id BIGINT UNSIGNED NULL,
    new_channel_id BIGINT UNSIGNED NULL,
    log_status VARCHAR(6) NOT NULL,
    created_at BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (id),
    KEY idx_voice_activity_logs_status_created_at (log_status, created_at),
    KEY idx_voice_activity_logs_user_created_at (user_id, created_at),
    CONSTRAINT chk_voice_activity_logs_status
        CHECK (log_status IN ('SENT', 'FAILED')),
    CONSTRAINT chk_voice_activity_logs_channels_present
        CHECK (old_channel_id IS NOT NULL OR new_channel_id IS NOT NULL),
    CONSTRAINT chk_voice_activity_logs_channels_differ
        CHECK (old_channel_id IS NULL OR new_channel_id IS NULL OR old_channel_id <> new_channel_id)
);

-- +goose Down
DROP TABLE voice_activity_logs;
