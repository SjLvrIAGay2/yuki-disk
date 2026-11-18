CREATE TABLE mounts(id text PRIMARY KEY,path text NOT NULL UNIQUE REFERENCES paths(path),node_id text NOT NULL REFERENCES nodes(id),export_id text NOT NULL);
CREATE TABLE visibility(path text PRIMARY KEY,mode text NOT NULL CHECK(mode IN ('public','private')),updated_at timestamptz NOT NULL DEFAULT now());
CREATE FUNCTION path_visibility_lifecycle() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' THEN DELETE FROM visibility WHERE path=OLD.path; RETURN OLD;
 ELSIF NEW.path<>OLD.path THEN UPDATE visibility SET path=NEW.path WHERE path=OLD.path; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER path_visibility AFTER UPDATE OF path OR DELETE ON paths FOR EACH ROW EXECUTE FUNCTION path_visibility_lifecycle();
INSERT INTO schema_migrations(version) VALUES(9);
