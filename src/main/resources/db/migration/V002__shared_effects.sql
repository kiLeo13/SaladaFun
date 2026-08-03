CREATE TABLE shared_effect_session (
    session_id TEXT PRIMARY KEY,
    session_label TEXT NOT NULL UNIQUE,
    session_date TEXT NOT NULL,
    daily_sequence INTEGER NOT NULL CHECK (daily_sequence > 0),
    status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'ARCHIVED')),
    initial_mode TEXT NOT NULL CHECK (
        initial_mode IN ('FRESH', 'SOURCE_PLAYER', 'RESUMED')
    ),
    source_player_uuid TEXT,
    revision INTEGER NOT NULL CHECK (revision >= 0),
    started_at TEXT NOT NULL,
    archived_at TEXT,
    UNIQUE (session_date, daily_sequence),
    CHECK ((initial_mode = 'SOURCE_PLAYER') = (source_player_uuid IS NOT NULL))
);

CREATE TABLE shared_effect_value (
    session_id TEXT NOT NULL REFERENCES shared_effect_session(session_id)
        ON DELETE CASCADE,
    effect_type TEXT NOT NULL,
    amplifier INTEGER NOT NULL CHECK (amplifier >= 0),
    duration_ticks INTEGER NOT NULL CHECK (
        duration_ticks = -1 OR duration_ticks > 0
    ),
    ambient INTEGER NOT NULL CHECK (ambient IN (0, 1)),
    particles INTEGER NOT NULL CHECK (particles IN (0, 1)),
    icon INTEGER NOT NULL CHECK (icon IN (0, 1)),
    PRIMARY KEY (session_id, effect_type)
);

CREATE TABLE shared_effect_control (
    singleton_id INTEGER PRIMARY KEY CHECK (singleton_id = 1),
    active_session_id TEXT REFERENCES shared_effect_session(session_id),
    updated_at TEXT NOT NULL
);

INSERT INTO shared_effect_control(singleton_id, active_session_id, updated_at)
VALUES (1, NULL, '1970-01-01T00:00:00Z');

CREATE TABLE player_effect_backup (
    session_id TEXT NOT NULL REFERENCES shared_effect_session(session_id)
        ON DELETE CASCADE,
    player_uuid TEXT NOT NULL,
    restore_status TEXT NOT NULL CHECK (
        restore_status IN ('CAPTURED', 'RESTORE_PENDING', 'RESTORED')
    ),
    captured_at TEXT NOT NULL,
    restored_at TEXT,
    PRIMARY KEY (session_id, player_uuid)
);

CREATE TABLE player_effect_backup_value (
    session_id TEXT NOT NULL,
    player_uuid TEXT NOT NULL,
    effect_type TEXT NOT NULL,
    amplifier INTEGER NOT NULL CHECK (amplifier >= 0),
    duration_ticks INTEGER NOT NULL CHECK (
        duration_ticks = -1 OR duration_ticks > 0
    ),
    ambient INTEGER NOT NULL CHECK (ambient IN (0, 1)),
    particles INTEGER NOT NULL CHECK (particles IN (0, 1)),
    icon INTEGER NOT NULL CHECK (icon IN (0, 1)),
    PRIMARY KEY (session_id, player_uuid, effect_type),
    FOREIGN KEY (session_id, player_uuid)
        REFERENCES player_effect_backup(session_id, player_uuid)
        ON DELETE CASCADE
);

CREATE INDEX idx_effect_backup_pending
    ON player_effect_backup(player_uuid, restore_status, captured_at DESC);

CREATE INDEX idx_effect_session_archived
    ON shared_effect_session(status, started_at DESC);
