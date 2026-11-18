CREATE TABLE inventory_sessions (
 node_id text PRIMARY KEY REFERENCES nodes(id), snapshot text NOT NULL,
 next_page integer NOT NULL DEFAULT 0, last_digest text NOT NULL DEFAULT '',
 completed boolean NOT NULL DEFAULT false, updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE inventory_entries (
 node_id text NOT NULL REFERENCES inventory_sessions(node_id) ON DELETE CASCADE,
 hash text NOT NULL, size bigint NOT NULL, generation bigint NOT NULL,
 PRIMARY KEY(node_id,hash,generation)
);
INSERT INTO schema_migrations(version) VALUES(5);
