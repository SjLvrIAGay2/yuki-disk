ALTER TABLE versions ALTER COLUMN path DROP NOT NULL;
ALTER TABLE versions ADD COLUMN trash_path text;
ALTER TABLE versions ADD COLUMN trashed_at timestamptz;
ALTER TABLE versions ADD COLUMN expires_at timestamptz;
ALTER TABLE versions ADD COLUMN trash_id text;
ALTER TABLE versions ADD CONSTRAINT version_trash_state CHECK (
 (path IS NOT NULL AND trash_path IS NULL AND trashed_at IS NULL AND expires_at IS NULL AND trash_id IS NULL)
 OR (path IS NULL AND trash_path IS NOT NULL AND trashed_at IS NOT NULL AND expires_at IS NOT NULL AND trash_id IS NOT NULL)
);
CREATE INDEX versions_trash_expiry ON versions(expires_at) WHERE path IS NULL;
CREATE INDEX versions_original_path ON versions(trash_path,id) WHERE path IS NULL;
INSERT INTO schema_migrations(version) VALUES(8);
