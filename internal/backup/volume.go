package backup

import (
	"fmt"
	"log/slog"
	"time"

	incus "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
)

func SnapshotVolume(logger *slog.Logger, source incus.InstanceServer, poolName, volumeName string) (*api.StorageVolume, error) {
	logger = logger.With("volume", volumeName)
	// 1. Check if Volume exists on Source pool
	incusVolume, _, err := source.GetStoragePoolVolume(poolName, "custom", volumeName)
	if err != nil {
		return nil, fmt.Errorf("volume not found on source: %w", err)
	}

	// 2. Create Snapshot
	snapshotName := fmt.Sprintf("IAB_%s", time.Now().Format("20060102-150405"))
	logger.Info("Creating snapshot", "snapshot", snapshotName)

	req := api.StorageVolumeSnapshotsPost{
		Name: snapshotName,
	}

	op, err := source.CreateStoragePoolVolumeSnapshot(poolName, "custom", volumeName, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}
	err = op.Wait()
	if err != nil {
		return nil, fmt.Errorf("snapshot operation failed: %w", err)
	}
	return incusVolume, nil
}

func CopyVolume(logger *slog.Logger, source, target incus.InstanceServer, poolName, volumeName, projectMode string, incusVolume *api.StorageVolume) error {
	logger = logger.With("volume", volumeName)
	// 3. Copy to target
	logger.Info("Copying volume to target")
	copyArgs := incus.StoragePoolVolumeCopyArgs{
		Name:       volumeName,
		Mode:       projectMode,
		VolumeOnly: false,
		Refresh:    true,
	}

	// the copy operation
	opCopy, err := target.CopyStoragePoolVolume(poolName, source, poolName, *incusVolume, &copyArgs)
	if err != nil {
		return fmt.Errorf("failed to start copy operation: %w", err)
	}

	if err := opCopy.Wait(); err != nil {
		return fmt.Errorf("copy operation failed: %w", err)
	}

	logger.Info("Volume sync successful")
	return nil
}
