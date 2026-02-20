package runner

import (
	"fmt"
	"time"

	"github.com/rbnhln/incusAutobackup/internal/backup"
)

type VolumeSnapshotTask struct {
	ProjectName string
	PoolName    string
	VolumeName  string
}

type VolumeCopyTask struct {
	ProjectName string
	PoolName    string
	VolumeName  string
	Mode        string
}

type VolumePruneTask struct {
	ProjectName  string
	PoolName     string
	VolumeName   string
	SourcePolicy string
	TargetPolicy string
}

func (t VolumeSnapshotTask) Name() string {
	return fmt.Sprintf("snapshot volume %s/%s (%s)", t.PoolName, t.VolumeName, t.ProjectName)
}

func (t VolumeSnapshotTask) Execute(x *ExecCtx) error {
	logger := x.Logger.With("project", t.ProjectName, "pool", t.PoolName, "volume", t.VolumeName)

	if x.DryRunCopy {
		logger.Info("dry-run: skipping snapshot")
		return nil
	}

	source := x.Source.UseProject(t.ProjectName)

	vol, err := backup.SnapshotVolume(logger, source, t.PoolName, t.VolumeName)
	if err != nil {
		return err
	}

	key := volumeKey(t.ProjectName, t.PoolName, t.VolumeName)
	x.VolumeSnapshots[key] = vol
	return nil
}

func (t VolumeCopyTask) Name() string {
	return fmt.Sprintf("copy volume %s/%s (%s)", t.PoolName, t.VolumeName, t.ProjectName)
}

func (t VolumeCopyTask) Execute(x *ExecCtx) error {
	logger := x.Logger.With("project", t.ProjectName, "pool", t.PoolName, "volume", t.VolumeName)

	if x.DryRunCopy {
		logger.Info("dry-run: skipping copy")
		return nil
	}

	key := volumeKey(t.ProjectName, t.PoolName, t.VolumeName)
	vol, ok := x.VolumeSnapshots[key]
	if !ok {
		logger.Warn("skipping copy: no snapshot was created (snapshot may have failed)")
		return fmt.Errorf("no snapshot result found for volume %s – snapshot phase likely failed", key)
	}

	source := x.Source.UseProject(t.ProjectName)
	target := x.Target.UseProject(t.ProjectName)

	return backup.CopyVolume(logger, source, target, t.PoolName, t.VolumeName, t.Mode, vol)
}

func (t VolumePruneTask) Name() string {
	return fmt.Sprintf("prune volume snapshots %s/%s (%s)", t.PoolName, t.VolumeName, t.ProjectName)
}

func (t VolumePruneTask) Execute(x *ExecCtx) error {
	logger := x.Logger.With("project", t.ProjectName, "pool", t.PoolName, "volume", t.VolumeName)

	source := x.Source.UseProject(t.ProjectName)
	target := x.Target.UseProject(t.ProjectName)

	now := time.Now()
	return backup.PruneVolume(logger, source, target, t.PoolName, t.VolumeName, t.SourcePolicy, t.TargetPolicy, now, x.DryRunPrune)
}

func volumeKey(project, pool, volume string) string {
	return fmt.Sprintf("%s/%s/%s", project, pool, volume)
}
