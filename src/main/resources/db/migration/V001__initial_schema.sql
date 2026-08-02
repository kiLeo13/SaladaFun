CREATE TABLE IF NOT EXISTS schema_history (
    version INTEGER PRIMARY KEY,
    description TEXT NOT NULL,
    checksum TEXT NOT NULL,
    installed_at TEXT NOT NULL
);

CREATE TABLE shared_health_session (
    session_id TEXT PRIMARY KEY,
    session_label TEXT NOT NULL UNIQUE,
    session_date TEXT NOT NULL,
    daily_sequence INTEGER NOT NULL CHECK (daily_sequence > 0),
    status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'ARCHIVED')),
    initial_mode TEXT NOT NULL CHECK (
        initial_mode IN ('FRESH', 'SOURCE_PLAYER', 'RESUMED')
    ),
    source_player_uuid TEXT,
    health REAL NOT NULL CHECK (health >= 0),
    maximum_health REAL NOT NULL CHECK (maximum_health > 0),
    absorption REAL NOT NULL CHECK (absorption >= 0),
    maximum_absorption REAL NOT NULL CHECK (maximum_absorption >= 0),
    phase TEXT NOT NULL CHECK (phase IN ('ALIVE', 'DEAD')),
    revision INTEGER NOT NULL CHECK (revision >= 0),
    started_at TEXT NOT NULL,
    archived_at TEXT,
    UNIQUE (session_date, daily_sequence),
    CHECK (health <= maximum_health),
    CHECK (absorption <= maximum_absorption),
    CHECK ((phase = 'DEAD') = (health = 0)),
    CHECK ((initial_mode = 'SOURCE_PLAYER') = (source_player_uuid IS NOT NULL))
);

CREATE TABLE shared_health_control (
    singleton_id INTEGER PRIMARY KEY CHECK (singleton_id = 1),
    active_session_id TEXT REFERENCES shared_health_session(session_id),
    updated_at TEXT NOT NULL
);

INSERT INTO shared_health_control(singleton_id, active_session_id, updated_at)
VALUES (1, NULL, '1970-01-01T00:00:00Z');

CREATE TABLE player_health_backup (
    session_id TEXT NOT NULL REFERENCES shared_health_session(session_id) ON DELETE CASCADE,
    player_uuid TEXT NOT NULL,
    health REAL NOT NULL,
    maximum_health REAL NOT NULL,
    absorption REAL NOT NULL,
    maximum_absorption REAL NOT NULL,
    phase TEXT NOT NULL CHECK (phase IN ('ALIVE', 'DEAD')),
    restore_status TEXT NOT NULL CHECK (
        restore_status IN ('CAPTURED', 'RESTORE_PENDING', 'RESTORED')
    ),
    captured_at TEXT NOT NULL,
    restored_at TEXT,
    PRIMARY KEY (session_id, player_uuid),
    CHECK (health >= 0 AND health <= maximum_health),
    CHECK (maximum_health > 0),
    CHECK (absorption >= 0 AND absorption <= maximum_absorption),
    CHECK (maximum_absorption >= 0),
    CHECK ((phase = 'DEAD') = (health = 0))
);

CREATE INDEX idx_health_backup_pending
    ON player_health_backup(player_uuid, restore_status, captured_at DESC);

CREATE INDEX idx_health_session_archived
    ON shared_health_session(status, started_at DESC);

CREATE TABLE shared_food_session (
    session_id TEXT PRIMARY KEY,
    session_label TEXT NOT NULL UNIQUE,
    session_date TEXT NOT NULL,
    daily_sequence INTEGER NOT NULL CHECK (daily_sequence > 0),
    status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'ARCHIVED')),
    initial_mode TEXT NOT NULL CHECK (
        initial_mode IN ('FRESH', 'SOURCE_PLAYER', 'RESUMED')
    ),
    source_player_uuid TEXT,
    food_level INTEGER NOT NULL CHECK (food_level BETWEEN 0 AND 20),
    saturation REAL NOT NULL CHECK (saturation >= 0),
    exhaustion REAL NOT NULL CHECK (exhaustion BETWEEN 0 AND 40),
    revision INTEGER NOT NULL CHECK (revision >= 0),
    started_at TEXT NOT NULL,
    archived_at TEXT,
    UNIQUE (session_date, daily_sequence),
    CHECK (saturation <= food_level),
    CHECK ((initial_mode = 'SOURCE_PLAYER') = (source_player_uuid IS NOT NULL))
);

CREATE TABLE shared_food_control (
    singleton_id INTEGER PRIMARY KEY CHECK (singleton_id = 1),
    active_session_id TEXT REFERENCES shared_food_session(session_id),
    updated_at TEXT NOT NULL
);

INSERT INTO shared_food_control(singleton_id, active_session_id, updated_at)
VALUES (1, NULL, '1970-01-01T00:00:00Z');

CREATE TABLE player_food_backup (
    session_id TEXT NOT NULL REFERENCES shared_food_session(session_id) ON DELETE CASCADE,
    player_uuid TEXT NOT NULL,
    food_level INTEGER NOT NULL CHECK (food_level BETWEEN 0 AND 20),
    saturation REAL NOT NULL CHECK (saturation >= 0),
    exhaustion REAL NOT NULL CHECK (exhaustion BETWEEN 0 AND 40),
    restore_status TEXT NOT NULL CHECK (
        restore_status IN ('CAPTURED', 'RESTORE_PENDING', 'RESTORED')
    ),
    captured_at TEXT NOT NULL,
    restored_at TEXT,
    PRIMARY KEY (session_id, player_uuid),
    CHECK (saturation <= food_level)
);

CREATE INDEX idx_food_backup_pending
    ON player_food_backup(player_uuid, restore_status, captured_at DESC);

CREATE INDEX idx_food_session_archived
    ON shared_food_session(status, started_at DESC);
