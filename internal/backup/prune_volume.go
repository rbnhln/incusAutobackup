package backup

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	incus "github.com/lxc/incus/v6/client"
	"github.com/rbnhln/incusAutobackup/internal/retention"
)

func PruneVolume(logger *slog.Logger, source, target incus.InstanceServer, poolName, volumeName, sourcePolicy, targetPolicy string, now time.Time, pruneDryRune bool) error {
	err := pruneVolumeSnapshots(logger, "source", source, poolName, volumeName, sourcePolicy, now, pruneDryRune)
	if err != nil {
		return fmt.Errorf("source prune failed: %w", err)
	}
	err = pruneVolumeSnapshots(logger, "target", target, poolName, volumeName, targetPolicy, now, pruneDryRune)
	if err != nil {
		return fmt.Errorf("target prune failed: %w", err)
	}
	return nil
}

func pruneVolumeSnapshots(
	logger *slog.Logger,
	role string,
	client incus.InstanceServer,
	poolName string,
	volumeName string,
	policy string,
	now time.Time,
	pruneDryRun bool,
) error {
	if strings.TrimSpace(policy) == "" {
		logger.Info("retention disabled; keeping all IAB snapshots",
			"role", role,
			"kind", "volume",
			"pool", poolName,
			"volume", volumeName,
		)
		return nil
	}

	ops := retention.SnapshotOps{
		Kind: "volume",
		List: func() ([]string, error) {
			snaps, err := client.GetStoragePoolVolumeSnapshots(poolName, "custom", volumeName)
			if err != nil {
				return nil, err
			}

			names := make([]string, 0, len(snaps))
			for _, s := range snaps {
				// Je nach Endpoint kann Name "volume/snap" sein
				n := s.Name
				if i := strings.LastIndex(n, "/"); i >= 0 && i < len(n)-1 {
					n = n[i+1:]
				}
				names = append(names, n)
			}
			return names, nil
		},
		Delete: func(name string) error {
			op, err := client.DeleteStoragePoolVolumeSnapshot(poolName, "custom", volumeName, name)
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
		logger.Warn("found IAB snapshots with timestamps in the future; keeping them",
			"role", role,
			"count", len(plan.Future),
		)
	}

	if len(plan.Remove) > 0 {
		logger.Info("prune result",
			"role", role,
			"kind", "volume",
			"pool", poolName,
			"volume", volumeName,
			"policy", policy,
			"dryRun", pruneDryRun,
			"keep", len(plan.Keep),
			"remove", len(plan.Remove),
			"unmanaged", len(plan.Unmanaged),
		)
	} else {
		logger.Debug("prune result (nothing to remove)",
			"role", role,
			"kind", "volume",
			"pool", poolName,
			"volume", volumeName,
			"policy", policy,
			"dryRun", pruneDryRun,
			"keep", len(plan.Keep),
			"remove", 0,
			"unmanaged", len(plan.Unmanaged),
		)
	}

	return nil
}
