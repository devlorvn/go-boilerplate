DROP INDEX IF EXISTS idx_files_owner_created_at;
ALTER TABLE files DROP COLUMN IF EXISTS owner_id;
