CREATE TABLE file_shares(
 id text PRIMARY KEY, owner_id text NOT NULL REFERENCES accounts(id), version_id text NOT NULL,
 kind text NOT NULL CHECK(kind IN ('user','api')), password_hash text NOT NULL DEFAULT '',
 expires_at timestamptz NOT NULL, revoked_at timestamptz, created_at timestamptz NOT NULL DEFAULT now(),
 idem text NOT NULL, request_digest text NOT NULL, UNIQUE(owner_id,idem)
);
CREATE TABLE api_applications(
 id text PRIMARY KEY, grant_id text NOT NULL REFERENCES file_shares(id), name text NOT NULL,
 credential_hash text NOT NULL UNIQUE, expires_at timestamptz NOT NULL, revoked_at timestamptz,
 created_by text NOT NULL REFERENCES accounts(id), idem text NOT NULL UNIQUE, request_digest text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE FUNCTION revoke_resource_shares() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_TABLE_NAME='versions' THEN
  IF TG_OP='DELETE' OR NEW.trashed_at IS NOT NULL THEN UPDATE file_shares SET revoked_at=coalesce(revoked_at,now()) WHERE version_id=OLD.id; END IF;
 ELSE
  IF NEW.trashed_at IS NOT NULL THEN UPDATE file_shares SET revoked_at=coalesce(revoked_at,now()) WHERE owner_id=OLD.owner_id AND version_id IN (SELECT id FROM versions WHERE owner_id=OLD.owner_id AND path=OLD.path); END IF;
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF; RETURN NEW;
END $$;
CREATE TRIGGER version_share_lifecycle BEFORE DELETE OR UPDATE OF trashed_at ON versions FOR EACH ROW EXECUTE FUNCTION revoke_resource_shares();
CREATE TRIGGER path_share_lifecycle BEFORE UPDATE OF trashed_at ON paths FOR EACH ROW EXECUTE FUNCTION revoke_resource_shares();
INSERT INTO schema_migrations(version) VALUES(11);
