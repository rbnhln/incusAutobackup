package runner

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/rbnhln/incusAutobackup/internal/retention"
	"github.com/rbnhln/incusAutobackup/internal/target"
	"github.com/rbnhln/incusAutobackup/internal/transfer"
)

func pruneTargetWithCtx(ctx context.Context, logger *slog.Logger, role string, t target.Target, kind transfer.Kind, subject, policy, projectName string, now time.Time, dryRun bool) error {
	if strings.TrimSpace(policy) == "" {
		logger.Info("retention disabled; keeping all IAB snapshots", "role", role, "kind", string(kind), "subject", subject, "target", t.Name())
		return nil
	}

	ops := retention.SnapshotOps{
		Kind: string(kind),
		List: func() ([]string, error) {
			points, err := t.List(ctx, kind, projectName, subject)
			if err != nil {
				return nil, err
			}
			names := make([]string, 0, len(points))
			for _, p := range points {
				names = append(names, p.Name)
			}
			return names, nil
		},
		Delete: func(name string) error {
			if dryRun {
				return nil
			}
			return t.Delete(ctx, projectName, transfer.RecoveryPoint{
				Kind:    kind,
				Project: projectName,
				Subject: subject,
				Name:    name,
			})
		},
	}

	plan, err := retention.PruneSnapshots(ops, policy, retention.PruneOptions{
		Now:     now,
		DryRun:  dryRun,
		Prefix:  retention.IABSnapshotPrefix,
		ParseTS: retention.ParseIABSnapshotTime,
	})
	if err != nil {
		return err
	}

	err = planLog(logger, plan, kind, subject, policy, role, dryRun)
	return err
}
