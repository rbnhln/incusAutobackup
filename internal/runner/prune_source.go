package runner

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	incus "github.com/lxc/incus/v6/client"
	"github.com/rbnhln/incusAutobackup/internal/retention"
	"github.com/rbnhln/incusAutobackup/internal/transfer"
)

func pruneSourceInstance(logger *slog.Logger, role string, client incus.InstanceServer, instanceName, policy string, now time.Time, dryRun bool) error {
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
		DryRun:  dryRun,
		Prefix:  retention.IABSnapshotPrefix,
		ParseTS: retention.ParseIABSnapshotTime,
	})
	if err != nil {
		return err
	}

	err = planLog(logger, plan, transfer.Kind("instance"), instanceName, policy, role, dryRun)
	return err
}

func pruneSourceVolume(logger *slog.Logger, role string, client incus.InstanceServer, poolName, volumeName, policy string, now time.Time, dryRun bool) error {
	if strings.TrimSpace(policy) == "" {
		logger.Info("retention disabled; keeping all IAB snapshots", "role", role, "kind", "volume", "pool", poolName, "volume", volumeName)
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
		DryRun:  dryRun,
		Prefix:  retention.IABSnapshotPrefix,
		ParseTS: retention.ParseIABSnapshotTime,
	})
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("%s/%s", poolName, volumeName)
	err = planLog(logger, plan, transfer.Kind("volume"), subject, policy, role, dryRun)
	return err
}
