-- Migration: Update-Based Telemetry Processing
-- Description: Modify schema to support hardware that updates existing rows instead of inserting new ones
-- Date: 2026-02-06

-- =============================================================================
-- STEP 1: Backup existing data
-- =============================================================================
-- Run this before migration:
-- mysqldump -u root -p iot_theater > backup_before_update_migration.sql

-- =============================================================================
-- STEP 2: Check for duplicate room names (IMPORTANT!)
-- =============================================================================
-- This query will show if you have duplicate room_name entries
-- You MUST resolve these before making room_name UNIQUE

SELECT room_name, COUNT(*) as count, GROUP_CONCAT(id) as ids
FROM theater_raw_telemetry
GROUP BY room_name
HAVING COUNT(*) > 1;

-- If duplicates exist, decide which row to keep for each room
-- Example: Keep the most recent row and delete others
-- DELETE FROM theater_raw_telemetry 
-- WHERE id IN (1, 2, 3);  -- Replace with actual duplicate IDs

-- =============================================================================
-- STEP 3: Modify theater_raw_telemetry table
-- =============================================================================

-- Add updated_at column to track when hardware updates the row
ALTER TABLE theater_raw_telemetry 
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
AFTER created_at;

-- Add index on room_name for faster lookups
CREATE INDEX IF NOT EXISTS idx_room_name ON theater_raw_telemetry(room_name);

-- Make room_name unique (one row per room)
-- NOTE: This will fail if duplicate room_name values exist
-- Make sure to run STEP 2 first!
ALTER TABLE theater_raw_telemetry 
ADD CONSTRAINT unique_room_name UNIQUE (room_name);

-- =============================================================================
-- STEP 4: Modify theater_live_state table
-- =============================================================================

-- Add last_processed_at column to track timestamp instead of ID
ALTER TABLE theater_live_state 
ADD COLUMN IF NOT EXISTS last_processed_at TIMESTAMP NULL 
AFTER last_processed_raw_id;

-- Add comment to mark last_processed_raw_id as deprecated
ALTER TABLE theater_live_state 
MODIFY COLUMN last_processed_raw_id INT DEFAULT 0 
COMMENT 'Deprecated: Use last_processed_at instead';

-- =============================================================================
-- STEP 5: Initialize data for existing rooms
-- =============================================================================

-- Update existing live states to set initial last_processed_at
-- This prevents the worker from reprocessing old data
UPDATE theater_live_state ls
INNER JOIN theater_raw_telemetry rt ON ls.room_name = rt.room_name
SET ls.last_processed_at = rt.updated_at
WHERE ls.last_processed_at IS NULL;

-- =============================================================================
-- STEP 6: Verify migration
-- =============================================================================

-- Check theater_raw_telemetry structure
DESCRIBE theater_raw_telemetry;

-- Check theater_live_state structure
DESCRIBE theater_live_state;

-- Verify unique constraint on room_name
SHOW INDEXES FROM theater_raw_telemetry WHERE Key_name = 'unique_room_name';

-- Check data
SELECT 
    rt.room_name,
    rt.created_at,
    rt.updated_at,
    ls.last_processed_at,
    ls.last_processed_raw_id
FROM theater_raw_telemetry rt
LEFT JOIN theater_live_state ls ON rt.room_name = ls.room_name;

-- =============================================================================
-- STEP 7: Add sample data for multiple rooms (optional - for testing)
-- =============================================================================

-- Insert/update telemetry for multiple rooms
INSERT INTO theater_raw_telemetry 
    (room_name, temp, humidity, room_pressure, room_status, laju_aliran_ahu, volume_ruangan, logic_ahu, oxygen, air, vacuum, instrument, carbon) 
VALUES 
    ('OT-01', 22.5, 55, 1.2, 1, 500, 100, 0, 95.0, 21.0, 1, 100.0, 100.0),
    ('OT-02', 23.0, 60, 1.1, 1, 450, 90, 0, 96.0, 21.0, 1, 100.0, 100.0),
    ('OT-03', 21.5, 50, 1.3, 1, 520, 110, 0, 94.0, 21.0, 1, 100.0, 100.0)
ON DUPLICATE KEY UPDATE 
    temp = VALUES(temp),
    humidity = VALUES(humidity),
    room_pressure = VALUES(room_pressure),
    updated_at = CURRENT_TIMESTAMP;

-- Initialize live states for all rooms
INSERT INTO theater_live_state (room_name) 
VALUES ('OT-01'), ('OT-02'), ('OT-03')
ON DUPLICATE KEY UPDATE room_name=room_name;

-- =============================================================================
-- ROLLBACK (if something goes wrong)
-- =============================================================================

-- To rollback the migration:
/*
-- Remove unique constraint
ALTER TABLE theater_raw_telemetry DROP INDEX unique_room_name;

-- Remove updated_at column
ALTER TABLE theater_raw_telemetry DROP COLUMN updated_at;

-- Remove index
DROP INDEX idx_room_name ON theater_raw_telemetry;

-- Remove last_processed_at column
ALTER TABLE theater_live_state DROP COLUMN last_processed_at;

-- Restore from backup
-- mysql -u root -p iot_theater < backup_before_update_migration.sql
*/

-- =============================================================================
-- NOTES
-- =============================================================================

/*
1. This migration changes the behavior from INSERT-based to UPDATE-based
2. Hardware should UPDATE the same row per room, not INSERT new rows
3. The worker will detect updates via the updated_at timestamp
4. Multiple rooms are now supported automatically
5. The worker polls every 500ms and checks all rooms

Hardware Update Pattern:
------------------------
UPDATE theater_raw_telemetry 
SET 
    temp = 23.5,
    humidity = 60,
    logic_ahu = 1,
    updated_at = NOW()
WHERE room_name = 'OT-01';

Worker Processing:
------------------
1. Fetch all raw telemetry (one row per room)
2. Fetch all live states
3. For each room: if raw.updated_at > live_state.last_processed_at
   - Process telemetry
   - Update live state
   - Set live_state.last_processed_at = raw.updated_at
*/
