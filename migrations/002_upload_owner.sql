ALTER TABLE blocks ADD COLUMN upload_owner text NOT NULL DEFAULT '';
INSERT INTO schema_migrations(version) VALUES(2);
