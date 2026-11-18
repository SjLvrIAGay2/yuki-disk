CREATE TABLE IF NOT EXISTS schema_migrations(version integer PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE nodes (
 id text PRIMARY KEY, url text UNIQUE NOT NULL, tier text NOT NULL CHECK(tier IN ('hot','cold')),
 last_seen timestamptz, inventory_at timestamptz
);
CREATE TABLE blocks (
 hash text PRIMARY KEY CHECK(hash ~ '^[0-9a-f]{64}$'), size bigint NOT NULL CHECK(size > 0 AND size <= 4194304),
 state text NOT NULL CHECK(state IN ('uploading','available','reclaimable','deleted')),
 generation bigint NOT NULL DEFAULT 1 CHECK(generation > 0), refs bigint NOT NULL DEFAULT 0 CHECK(refs>=0),
 upload_until timestamptz, gc_after timestamptz, last_access timestamptz NOT NULL DEFAULT now(),
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX blocks_gc ON blocks(gc_after) WHERE state='reclaimable';
CREATE TABLE replicas (
 hash text NOT NULL REFERENCES blocks(hash), node_id text NOT NULL REFERENCES nodes(id),
 generation bigint NOT NULL, confirmed_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(hash,node_id)
);
CREATE TABLE paths (path text PRIMARY KEY, trashed_at timestamptz, expires_at timestamptz);
CREATE TABLE uploads (
 id text PRIMARY KEY, idem text UNIQUE NOT NULL, request_digest text NOT NULL, path text NOT NULL,
 manifest jsonb NOT NULL, size bigint NOT NULL CHECK(size >= 0), expires_at timestamptz NOT NULL,
 committed boolean NOT NULL DEFAULT false, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE versions (
 id text PRIMARY KEY REFERENCES uploads(id), path text NOT NULL REFERENCES paths(path),
 size bigint NOT NULL CHECK(size>=0), created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX versions_path ON versions(path,created_at,id);
CREATE TABLE version_blocks (
 version_id text NOT NULL REFERENCES versions(id) ON DELETE CASCADE, ordinal integer NOT NULL CHECK(ordinal>=0),
 hash text NOT NULL REFERENCES blocks(hash), PRIMARY KEY(version_id,ordinal)
);
CREATE INDEX version_blocks_hash ON version_blocks(hash);
CREATE TABLE replay_nonces (identity text NOT NULL, nonce text NOT NULL, expires_at timestamptz NOT NULL, PRIMARY KEY(identity,nonce));
CREATE INDEX replay_expiry ON replay_nonces(expires_at);
CREATE TABLE leader (
 singleton boolean PRIMARY KEY DEFAULT true CHECK(singleton), owner text NOT NULL DEFAULT '',
 fence bigint NOT NULL DEFAULT 0, expires_at timestamptz NOT NULL DEFAULT '-infinity'
);
INSERT INTO leader(singleton) VALUES(true);
CREATE TABLE tasks (
 id bigserial PRIMARY KEY, kind text NOT NULL CHECK(kind IN ('copy','delete','inventory')),
 hash text, node_id text NOT NULL REFERENCES nodes(id), generation bigint NOT NULL DEFAULT 0,
 payload jsonb NOT NULL DEFAULT '{}', state text NOT NULL DEFAULT 'queued' CHECK(state IN ('queued','running','done','failed')),
 owner text, fence bigint, attempts integer NOT NULL DEFAULT 0,
 available_at timestamptz NOT NULL DEFAULT now(), lease_until timestamptz,
 error text NOT NULL DEFAULT '', created_at timestamptz NOT NULL DEFAULT now(), finished_at timestamptz
);
CREATE UNIQUE INDEX tasks_active ON tasks(kind,coalesce(hash,''),node_id,generation) WHERE state IN ('queued','running');
CREATE INDEX tasks_ready ON tasks(available_at,id) WHERE state='queued';
CREATE TABLE events (id bigserial PRIMARY KEY, kind text NOT NULL, hash text, detail jsonb NOT NULL DEFAULT '{}', created_at timestamptz NOT NULL DEFAULT now());
CREATE INDEX events_created ON events(created_at);
INSERT INTO schema_migrations(version) VALUES(1);
