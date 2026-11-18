CREATE TABLE node_resources (
 node_id text PRIMARY KEY REFERENCES nodes(id) ON DELETE CASCADE,
 sampled_at timestamptz NOT NULL, received_at timestamptz NOT NULL DEFAULT now(),
 boot_id text NOT NULL, sequence bigint NOT NULL CHECK(sequence>0),
 payload jsonb NOT NULL CHECK(jsonb_typeof(payload)='object')
);
ALTER TABLE nodes ADD COLUMN read_until timestamptz NOT NULL DEFAULT 'epoch';
ALTER TABLE nodes ADD COLUMN write_until timestamptz NOT NULL DEFAULT 'epoch';
ALTER TABLE nodes ADD COLUMN read_bandwidth_bps bigint NOT NULL DEFAULT 0 CHECK(read_bandwidth_bps>=0);
ALTER TABLE nodes ADD COLUMN write_bandwidth_bps bigint NOT NULL DEFAULT 0 CHECK(write_bandwidth_bps>=0);
ALTER TABLE nodes ADD COLUMN reserve_bytes bigint NOT NULL DEFAULT 0 CHECK(reserve_bytes>=0);
ALTER TABLE blocks ADD COLUMN heat double precision NOT NULL DEFAULT 0 CHECK(heat>=0 AND heat<=1000000);
ALTER TABLE blocks ADD COLUMN heat_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE blocks ADD COLUMN last_move_at timestamptz;
CREATE TABLE placement_reservations (
 hash text NOT NULL REFERENCES blocks(hash) ON DELETE CASCADE,
 node_id text NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
 generation bigint NOT NULL CHECK(generation>0), size bigint NOT NULL CHECK(size>0 AND size<=4194304),
 created_at timestamptz NOT NULL DEFAULT now(), expires_at timestamptz NOT NULL,
 PRIMARY KEY(hash,node_id,generation)
);
CREATE INDEX placement_node ON placement_reservations(node_id,expires_at);
CREATE INDEX placement_expiry ON placement_reservations(expires_at);
CREATE INDEX replicas_node ON replicas(node_id);
INSERT INTO schema_migrations(version) VALUES(7);
