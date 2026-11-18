CREATE TABLE runtime_configuration(singleton boolean PRIMARY KEY DEFAULT true CHECK(singleton),version bigint NOT NULL,settings jsonb NOT NULL,updated_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE configuration_instances(id text PRIMARY KEY,target_version bigint NOT NULL,applied_version bigint NOT NULL,state text NOT NULL CHECK(state IN ('applied','failed')),observed_at timestamptz NOT NULL DEFAULT now());
INSERT INTO schema_migrations(version) VALUES(12);
