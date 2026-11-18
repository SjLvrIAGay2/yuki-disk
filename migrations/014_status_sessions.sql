ALTER TABLE sessions DROP CONSTRAINT sessions_client_check;
ALTER TABLE sessions ADD CONSTRAINT sessions_client_check CHECK(client IN ('web','cli','status'));
INSERT INTO schema_migrations(version) VALUES(14);
