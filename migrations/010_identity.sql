CREATE TABLE accounts (
 id text PRIMARY KEY, username text NOT NULL UNIQUE, password_hash text NOT NULL,
 role text NOT NULL CHECK(role IN ('admin','user')), disabled boolean NOT NULL DEFAULT false,
 auth_epoch bigint NOT NULL DEFAULT 1, created_at timestamptz NOT NULL DEFAULT now()
);
-- Reserved, inaccessible until the operator explicitly names the legacy owner.
INSERT INTO accounts(id,username,password_hash,role,disabled) VALUES('legacy','!unassigned-legacy','','user',true);
CREATE TABLE sessions (
 id text PRIMARY KEY, user_id text NOT NULL REFERENCES accounts(id), auth_epoch bigint NOT NULL,
 client text NOT NULL CHECK(client IN ('web','cli')), refresh_hash text UNIQUE,
 previous_hash text, previous_until timestamptz, rotation bigint NOT NULL DEFAULT 0,
 expires_at timestamptz NOT NULL, idle_until timestamptz NOT NULL,
 revoked_at timestamptz, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX sessions_previous_hash ON sessions(previous_hash) WHERE previous_hash IS NOT NULL;
CREATE TABLE security_audit(id bigserial PRIMARY KEY, actor_id text REFERENCES accounts(id), action text NOT NULL, target_id text, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE auth_attempts(key text PRIMARY KEY, count integer NOT NULL, expires_at timestamptz NOT NULL);
CREATE FUNCTION request_owner() RETURNS text LANGUAGE sql STABLE AS $$ SELECT coalesce(nullif(current_setting('yuki.owner',true),''),'legacy') $$;
ALTER TABLE paths ADD COLUMN owner_id text NOT NULL DEFAULT request_owner() REFERENCES accounts(id);
ALTER TABLE uploads ADD COLUMN owner_id text NOT NULL DEFAULT request_owner() REFERENCES accounts(id);
ALTER TABLE versions ADD COLUMN owner_id text NOT NULL DEFAULT request_owner() REFERENCES accounts(id);
ALTER TABLE operations ADD COLUMN owner_id text NOT NULL DEFAULT request_owner() REFERENCES accounts(id);
ALTER TABLE visibility ADD COLUMN owner_id text NOT NULL DEFAULT request_owner() REFERENCES accounts(id);
ALTER TABLE mounts ADD COLUMN owner_id text NOT NULL DEFAULT request_owner() REFERENCES accounts(id);
ALTER TABLE versions DROP CONSTRAINT versions_path_fkey;
ALTER TABLE mounts DROP CONSTRAINT mounts_path_fkey;
ALTER TABLE paths DROP CONSTRAINT paths_pkey;
ALTER TABLE paths ADD PRIMARY KEY(owner_id,path);
ALTER TABLE versions ADD FOREIGN KEY(owner_id,path) REFERENCES paths(owner_id,path) ON UPDATE CASCADE;
ALTER TABLE mounts ADD FOREIGN KEY(owner_id,path) REFERENCES paths(owner_id,path) ON UPDATE CASCADE;
ALTER TABLE mounts DROP CONSTRAINT mounts_path_key;
ALTER TABLE mounts ADD UNIQUE(owner_id,path);
ALTER TABLE uploads DROP CONSTRAINT uploads_idem_key;
ALTER TABLE uploads ADD UNIQUE(owner_id,idem);
ALTER TABLE operations DROP CONSTRAINT operations_pkey;
ALTER TABLE operations ADD PRIMARY KEY(owner_id,key);
ALTER TABLE visibility DROP CONSTRAINT visibility_pkey;
ALTER TABLE visibility ADD PRIMARY KEY(owner_id,path);
ALTER TABLE uploads ADD UNIQUE(owner_id,id);
ALTER TABLE versions ADD FOREIGN KEY(owner_id,id) REFERENCES uploads(owner_id,id);
CREATE OR REPLACE FUNCTION path_visibility_lifecycle() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' THEN DELETE FROM visibility WHERE owner_id=OLD.owner_id AND path=OLD.path; RETURN OLD;
 ELSIF NEW.path<>OLD.path THEN UPDATE visibility SET path=NEW.path WHERE owner_id=OLD.owner_id AND path=OLD.path; END IF;
 RETURN NEW;
END $$;
DO $$ DECLARE tab text; BEGIN
 FOREACH tab IN ARRAY ARRAY['paths','uploads','versions','operations','visibility','mounts'] LOOP
  EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY',tab);
  EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY',tab);
  EXECUTE format('CREATE POLICY owner_boundary ON %I USING (coalesce(current_setting(''yuki.owner'',true),'''')='''' OR owner_id=current_setting(''yuki.owner'',true)) WITH CHECK (coalesce(current_setting(''yuki.owner'',true),'''')='''' OR owner_id=current_setting(''yuki.owner'',true))',tab);
 END LOOP;
END $$;
INSERT INTO schema_migrations(version) VALUES(10);
