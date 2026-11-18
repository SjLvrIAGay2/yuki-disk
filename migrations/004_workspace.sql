ALTER TABLE paths ADD COLUMN is_dir boolean NOT NULL DEFAULT false;
ALTER TABLE paths ADD COLUMN created_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE versions DROP CONSTRAINT versions_path_fkey;
ALTER TABLE versions ADD CONSTRAINT versions_path_fkey FOREIGN KEY(path) REFERENCES paths(path) ON UPDATE CASCADE;
INSERT INTO paths(path,is_dir)
SELECT DISTINCT array_to_string(parts[1:i],'/'),true
FROM (SELECT string_to_array(path,'/') parts FROM paths) p,
LATERAL generate_series(1,array_length(parts,1)-1) i
ON CONFLICT DO NOTHING;
CREATE TABLE operations (
 key text PRIMARY KEY, request_digest text NOT NULL, result jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX paths_parent ON paths ((regexp_replace(path,'/?[^/]+$','')));
INSERT INTO schema_migrations(version) VALUES(4);
