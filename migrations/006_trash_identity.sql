ALTER TABLE paths ADD COLUMN trash_id text;
UPDATE paths SET trash_id=md5(path || ':' || trashed_at::text) WHERE trashed_at IS NOT NULL;
CREATE UNIQUE INDEX paths_trash_identity ON paths(trash_id) WHERE trash_id IS NOT NULL;
INSERT INTO schema_migrations(version) VALUES(6);
