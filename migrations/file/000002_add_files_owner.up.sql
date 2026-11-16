-- Files created before this migration have no owner; only admins can see them.
ALTER TABLE files ADD COLUMN IF NOT EXISTS owner_id UUID;
CREATE INDEX IF NOT EXISTS idx_files_owner_created_at ON files (owner_id, created_at DESC);
