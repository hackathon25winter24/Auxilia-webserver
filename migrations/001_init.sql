CREATE TABLE IF NOT EXISTS web_guests (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(80) NOT NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    selection_json TEXT NOT NULL,
    match_id VARCHAR(64) NOT NULL DEFAULT '',
    queued BOOLEAN NOT NULL DEFAULT FALSE,
    queued_at DATETIME(3) NULL,
    expires_at DATETIME(3) NOT NULL,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    INDEX idx_web_guests_match_id (match_id),
    INDEX idx_queue (queued, queued_at),
    INDEX idx_web_guests_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS web_matches (
    id VARCHAR(64) PRIMARY KEY,
    revision BIGINT UNSIGNED NOT NULL,
    status VARCHAR(16) NOT NULL,
    state_json LONGTEXT NOT NULL,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    INDEX idx_web_matches_status (status),
    CONSTRAINT chk_web_matches_state_json CHECK (JSON_VALID(state_json))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS web_processed_commands (
    id VARCHAR(180) PRIMARY KEY,
    match_id VARCHAR(64) NOT NULL,
    created_at DATETIME(3) NOT NULL,
    INDEX idx_web_processed_commands_match_id (match_id),
    INDEX idx_web_processed_commands_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
