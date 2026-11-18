package main

import (
	"context"
	"github.com/jackc/pgx/v5"
)

func upgradeStore(ctx context.Context, t pgx.Tx, version int) error {
	upgrades := []string{"002_upload_owner.sql", "003_task_kinds.sql", "004_workspace.sql", "005_inventory_pages.sql", "006_trash_identity.sql", "007_adaptive_routing.sql", "008_version_trash.sql", "009_mounts_visibility.sql", "010_identity.sql", "011_shares.sql", "012_runtime.sql", "013_upload_proofs.sql", "014_status_sessions.sql", "015_administration.sql", "016_runtime_logs.sql", "017_log_policy_observation.sql"}
	for index, name := range upgrades {
		if version >= index+2 {
			continue
		}
		b, err := migrations.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		if _, err = t.Exec(ctx, string(b)); err != nil {
			return err
		}
	}
	return nil
}
