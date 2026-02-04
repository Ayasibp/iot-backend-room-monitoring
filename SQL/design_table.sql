-- 1. Users & RBAC
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role ENUM('admin', 'user') DEFAULT 'user',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. Refresh Tokens (Security: Revocable Sessions)
CREATE TABLE refresh_tokens (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    revoked BOOLEAN DEFAULT FALSE, 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 3. RAW TELEMETRY (Hardware writes here)
-- The hardware inserts data here. It does NOT perform calculations.
CREATE TABLE theater_raw_telemetry (
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
CREATE TABLE theater_live_state (
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
CREATE TABLE audit_logs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT,
    action VARCHAR(100) NOT NULL, 
    details TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);