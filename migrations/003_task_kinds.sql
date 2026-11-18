ALTER TABLE tasks DROP CONSTRAINT tasks_kind_check;
ALTER TABLE tasks ADD CONSTRAINT tasks_kind_check CHECK(kind IN ('copy','delete'));
INSERT INTO schema_migrations(version) VALUES(3);
