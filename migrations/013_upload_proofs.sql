CREATE TABLE upload_proofs(upload_id text NOT NULL REFERENCES uploads(id) ON DELETE CASCADE,hash text NOT NULL REFERENCES blocks(hash),generation bigint NOT NULL,created_at timestamptz NOT NULL DEFAULT now(),PRIMARY KEY(upload_id,hash));
INSERT INTO schema_migrations(version) VALUES(13);
