ALTER TABLE log_streams ADD COLUMN policy_version bigint NOT NULL DEFAULT 0;
INSERT INTO schema_migrations(version) VALUES(17);
