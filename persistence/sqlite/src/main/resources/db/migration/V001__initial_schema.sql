CREATE TABLE IF NOT EXISTS schema_history (
    version INTEGER PRIMARY KEY,
    description TEXT NOT NULL,
    checksum TEXT NOT NULL,
    installed_at TEXT NOT NULL
);

CREATE TABLE shared_inventory_session (
    session_id TEXT PRIMARY KEY,
    session_label TEXT NOT NULL UNIQUE,
    session_date TEXT NOT NULL,
    daily_sequence INTEGER NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'ARCHIVED')),
    initial_mode TEXT NOT NULL CHECK (initial_mode IN ('EMPTY', 'SOURCE_PLAYER', 'RESUMED')),
    source_player_uuid TEXT,
    revision INTEGER NOT NULL CHECK (revision >= 0),
    started_at TEXT NOT NULL,
    archived_at TEXT,
    UNIQUE (session_date, daily_sequence)
);

CREATE TABLE shared_inventory_control (
    singleton_id INTEGER PRIMARY KEY CHECK (singleton_id = 1),
    active_session_id TEXT REFERENCES shared_inventory_session(session_id),
    updated_at TEXT NOT NULL
);

INSERT INTO shared_inventory_control(singleton_id, active_session_id, updated_at)
VALUES (1, NULL, '1970-01-01T00:00:00Z');

CREATE TABLE shared_inventory_slot (
    session_id TEXT NOT NULL REFERENCES shared_inventory_session(session_id) ON DELETE CASCADE,
    slot_key TEXT NOT NULL,
    item_key TEXT NOT NULL,
    stacking_fingerprint TEXT NOT NULL,
    item_amount INTEGER NOT NULL CHECK (item_amount > 0),
    maximum_stack_size INTEGER NOT NULL CHECK (maximum_stack_size > 0),
    payload_format TEXT NOT NULL,
    item_payload BLOB NOT NULL,
    slot_revision INTEGER NOT NULL CHECK (slot_revision >= 0),
    PRIMARY KEY (session_id, slot_key)
);

CREATE TABLE player_inventory_backup (
    session_id TEXT NOT NULL REFERENCES shared_inventory_session(session_id) ON DELETE CASCADE,
    player_uuid TEXT NOT NULL,
    restore_status TEXT NOT NULL CHECK (
        restore_status IN ('CAPTURED', 'RESTORE_PENDING', 'RESTORED')
    ),
    captured_at TEXT NOT NULL,
    restored_at TEXT,
    PRIMARY KEY (session_id, player_uuid)
);

CREATE TABLE player_inventory_backup_slot (
    session_id TEXT NOT NULL,
    player_uuid TEXT NOT NULL,
    slot_key TEXT NOT NULL,
    item_key TEXT NOT NULL,
    stacking_fingerprint TEXT NOT NULL,
    item_amount INTEGER NOT NULL CHECK (item_amount > 0),
    maximum_stack_size INTEGER NOT NULL CHECK (maximum_stack_size > 0),
    payload_format TEXT NOT NULL,
    item_payload BLOB NOT NULL,
    PRIMARY KEY (session_id, player_uuid, slot_key),
    FOREIGN KEY (session_id, player_uuid)
        REFERENCES player_inventory_backup(session_id, player_uuid) ON DELETE CASCADE
);

CREATE TABLE player_inventory_replica (
    session_id TEXT NOT NULL REFERENCES shared_inventory_session(session_id) ON DELETE CASCADE,
    player_uuid TEXT NOT NULL,
    last_applied_revision INTEGER NOT NULL CHECK (last_applied_revision >= 0),
    last_inventory_fingerprint TEXT NOT NULL,
    observed_at TEXT NOT NULL,
    PRIMARY KEY (session_id, player_uuid)
);

CREATE INDEX idx_backup_pending
    ON player_inventory_backup(player_uuid, restore_status);

CREATE INDEX idx_session_archived
    ON shared_inventory_session(status, started_at DESC);
