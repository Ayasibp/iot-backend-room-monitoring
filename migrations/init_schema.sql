-- IoT Theater Monitoring System - Database Schema
-- MySQL Database Initialization Script

-- 1. Users & RBAC
CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role ENUM('admin', 'user') DEFAULT 'user',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. Refresh Tokens (Security: Revocable Sessions)
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    revoked BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_token_hash (token_hash),
    INDEX idx_user_id (user_id)
);

-- 3. RAW TELEMETRY (Hardware writes here)
-- The hardware inserts data here. It does NOT perform calculations.
CREATE TABLE IF NOT EXISTS theater_raw_telemetry (
    id INT AUTO_INCREMENT PRIMARY KEY,
    room_name VARCHAR(50) DEFAULT 'OT-01',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Inputs from Hardware
    temp FLOAT DEFAULT NULL,
    humidity INT DEFAULT NULL,
    room_pressure FLOAT DEFAULT NULL,
    room_status INT DEFAULT 0, -- 0=Off, 1=On
    
    -- ACH Calculation Inputs
    laju_aliran_ahu INT DEFAULT 0,  -- Flow Rate
    volume_ruangan INT DEFAULT 0,   -- Room Volume
    logic_ahu INT DEFAULT 0,        -- The Trigger (0 or 1)
    
    -- Medical Gases
    oxygen FLOAT DEFAULT NULL,
    nitrous FLOAT DEFAULT NULL,
    air FLOAT DEFAULT NULL,
    vacuum INT DEFAULT NULL,
    instrument FLOAT DEFAULT NULL,
    carbon FLOAT DEFAULT NULL
);

-- 4. LIVE STATE (Dashboard reads this)
-- The Go Background Worker updates this single row.
CREATE TABLE IF NOT EXISTS theater_live_state (
    id INT AUTO_INCREMENT PRIMARY KEY,
    room_name VARCHAR(50) UNIQUE DEFAULT 'OT-01',
    
    -- A. Calculated Results (From Worker)
    ach_theoretical FLOAT DEFAULT 0.0, -- Method 1
    ach_empirical FLOAT DEFAULT 0.0,   -- Method 2
    
    -- B. Latest Sensor Values (Copied from Raw)
    current_temp FLOAT DEFAULT 0.0,
    current_pressure FLOAT DEFAULT 0.0,
    current_logic_ahu INT DEFAULT 0,
    
    -- C. Stopwatch Logic (Admin Controlled)
    op_start_time DATETIME NULL,
    op_accumulated_seconds INT DEFAULT 0,
    op_is_running TINYINT(1) DEFAULT 0,

    -- D. Countdown Logic (Admin Controlled)
    cd_target_time DATETIME NULL,
    cd_duration_seconds INT DEFAULT 3600,
    cd_is_running TINYINT(1) DEFAULT 0,
    
    -- E. Internal Worker State (To track Method 2 timing)
    ahu_cycle_start_time DATETIME NULL,  -- If NOT NULL, cycle is active
    last_processed_raw_id INT DEFAULT 0, -- Pointer to raw table
    
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 5. Audit Logs (Security)
CREATE TABLE IF NOT EXISTS audit_logs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT,
    action VARCHAR(100) NOT NULL,
    details TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_id (user_id)
);

-- ============================================
-- SEED DATA
-- ============================================

-- Insert default admin user
-- Password: admin123 (bcrypt hash with cost 12)
INSERT INTO users (username, password_hash, role) VALUES
('admin', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5S0FZb6Hxwunu', 'admin')
ON DUPLICATE KEY UPDATE username=username;

-- Insert default user
-- Password: user123 (bcrypt hash with cost 12)
INSERT INTO users (username, password_hash, role) VALUES
('user', '$2a$12$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'user')
ON DUPLICATE KEY UPDATE username=username;

-- Initialize theater live state for OT-01
INSERT INTO theater_live_state (room_name) VALUES ('OT-01')
ON DUPLICATE KEY UPDATE room_name=room_name;

-- Insert sample raw telemetry data (optional - for testing)
-- INSERT INTO theater_raw_telemetry 
-- (room_name, temp, humidity, room_pressure, room_status, laju_aliran_ahu, volume_ruangan, logic_ahu, oxygen, air) 
-- VALUES 
-- ('OT-01', 22.5, 55, 1.2, 1, 500, 100, 1, 95.0, 21.0);
