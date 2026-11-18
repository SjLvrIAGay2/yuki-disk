ALTER TABLE accounts ADD COLUMN deleted_at timestamptz;
ALTER TABLE accounts ADD COLUMN purge_after timestamptz;
ALTER TABLE accounts ADD COLUMN purged_at timestamptz;
ALTER TABLE file_shares ALTER COLUMN expires_at DROP NOT NULL;
ALTER TABLE file_shares ADD CHECK(kind='api' OR expires_at IS NOT NULL);
ALTER TABLE api_applications ALTER COLUMN expires_at DROP NOT NULL;
CREATE TABLE mount_sources(
 id text PRIMARY KEY, name text NOT NULL, node_id text NOT NULL REFERENCES nodes(id),
 export_id text NOT NULL, source_path text NOT NULL DEFAULT '', version bigint NOT NULL DEFAULT 1,
 created_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO mount_sources(id,name,node_id,export_id) SELECT id,path,node_id,export_id FROM mounts;
INSERT INTO security_audit(action,target_id) SELECT 'mount.seed',id FROM mount_sources;
ALTER TABLE mounts ADD COLUMN source_id text REFERENCES mount_sources(id);
ALTER TABLE mounts ADD COLUMN source_path text NOT NULL DEFAULT '';
UPDATE mounts SET source_id=id;
ALTER TABLE mounts ALTER COLUMN source_id SET NOT NULL;
UPDATE runtime_configuration SET settings='{"user_share_seconds":86400,"api_share_seconds":86400,"login_attempts":20,"download_authorizations":600}'::jsonb || settings;
INSERT INTO schema_migrations(version) VALUES(15);
