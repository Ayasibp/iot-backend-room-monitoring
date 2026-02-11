-- Device API Keys Migration
-- Creates table for storing ESP32 device API keys
-- Each room can have one or more API keys for authentication

CREATE TABLE IF NOT EXISTS device_api_keys (
    id INT AUTO_INCREMENT PRIMARY KEY,
    room_id INT NOT NULL,
    api_key_hash VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NULL,
    is_active TINYINT(1) DEFAULT 1,
    description VARCHAR(255) DEFAULT NULL COMMENT 'Optional description for the key (e.g., "ESP32 Device 1")',
    
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    INDEX idx_room_id (room_id),
    INDEX idx_api_key_hash (api_key_hash),
    INDEX idx_is_active (is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Sample comment for clarity
-- API keys are stored as bcrypt hashes for security
-- The plain-text key is only shown once when generated
