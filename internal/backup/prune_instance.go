package backup

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	incus "github.com/lxc/incus/v6/client"
	"github.com/rbnhln/incusAutobackup/internal/retention"
)

func PruneInstance(logger *slog.Logger, source, target incus.InstanceServer, instanceName, sourcePolicy, targetPolicy string, now time.Time, pruneDryRun bool) error {
	err := pruneInstanceSnapshots(logger, "source", source, instanceName, sourcePolicy, now, pruneDryRun)
	if err != nil {
		return fmt.Errorf("source prune failed: %w", err)
	}

	err = pruneInstanceSnapshots(logger, "target", target, instanceName, targetPolicy, now, pruneDryRun)
	if err != nil {
		return fmt.Errorf("target prune failed: %w", err)
	}
	return nil
}

func pruneInstanceSnapshots(logger *slog.Logger, role string, client incus.InstanceServer, instanceName, policy string, now time.Time, pruneDryRun bool) error {
	if strings.TrimSpace(policy) == "" {
		logger.Info("retention disabled; keeping all IAB snapshots", "role", role, "kind", "instance", "instance", instanceName)
		return nil
	}

	ops := retention.SnapshotOps{
		Kind: "instance",
		List: func() ([]string, error) {
			names, err := client.GetInstanceSnapshotNames(instanceName)
			if err != nil {
				return nil, err
			}

			normalized := make([]string, 0, len(names))
			for _, n := range names {
				if i := strings.LastIndex(n, "/"); i >= 0 && i < len(n)-1 {
					n = n[i+1:]
				}
				normalized = append(normalized, n)
			}
			return normalized, nil
		},
		Delete: func(name string) error {
			op, err := client.DeleteInstanceSnapshot(instanceName, name)
			if err != nil {
				return err
			}
			return op.Wait()
		},
	}

	plan, err := retention.PruneSnapshots(ops, policy, retention.PruneOptions{
		Now:     now,
		DryRun:  pruneDryRun,
		Prefix:  retention.IABSnapshotPrefix,
		ParseTS: retention.ParseIABSnapshotTime,
	})
	if err != nil {
		return err
	}

	if len(plan.Future) > 0 {
		logger.Warn("found IAB snapshots with timestamps in the future, keeping them", "role", role, "count", len(plan.Future))
	}

	if len(plan.Remove) > 0 {
		logger.Info("prune result",
			"role", role,
			"kind", "instance",
			"instance", instanceName,
			"policy", policy,
			"dryRun", pruneDryRun,
			"keep", len(plan.Keep),
			"remove", len(plan.Remove),
			"unmanaged", len(plan.Unmanaged),
		)
	} else {
		logger.Debug("prune result (nothing to remove)",
			"role", role,
			"kind", "instance",
			"instance", instanceName,
			"policy", policy,
			"dryRun", pruneDryRun,
			"keep", len(plan.Keep),
			"remove", 0,
			"unmanaged", len(plan.Unmanaged),
		)
	}
	return nil
}
