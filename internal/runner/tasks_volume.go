package runner

import (
	"fmt"
	"time"

	"github.com/rbnhln/incusAutobackup/internal/pipeline"
	"github.com/rbnhln/incusAutobackup/internal/target"
	"github.com/rbnhln/incusAutobackup/internal/transfer"
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
	ctx := x.Ctx
	logger := x.Logger.With("project", t.ProjectName, "pool", t.PoolName, "volume", t.VolumeName)

	if x.DryRunCopy {
		logger.Info("dry-run: skipping snapshot")
		return nil
	}

	arti, err := x.Source.PrepareVolume(ctx, t.ProjectName, t.PoolName, t.VolumeName)
	if err != nil {
		return err
	}

	key := volumeKey(t.ProjectName, t.PoolName, t.VolumeName)
	x.VolumeSnapshots[key] = arti
	return nil
}

func (t VolumeCopyTask) Name() string {
	return fmt.Sprintf("copy volume %s/%s (%s)", t.PoolName, t.VolumeName, t.ProjectName)
}

func (t VolumeCopyTask) Execute(x *ExecCtx) error {
	ctx := x.Ctx
	logger := x.Logger.With("project", t.ProjectName, "pool", t.PoolName, "volume", t.VolumeName)

	if x.DryRunCopy {
		logger.Info("dry-run: skipping copy")
		return nil
	}

	key := volumeKey(t.ProjectName, t.PoolName, t.VolumeName)
	arti, ok := x.VolumeSnapshots[key]
	if !ok {
		logger.Warn("skipping copy: no snapshot was created (snapshot may have failed)")
		return fmt.Errorf("no snapshot result found for volume %s – snapshot phase likely failed", key)
	}

	incusOpts := target.IncusCopyOptions{
		Mode: t.Mode,
	}

	return pipeline.Store(ctx, logger, x.Source, x.Target, arti, incusOpts)
}

func (t VolumePruneTask) Name() string {
	return fmt.Sprintf("prune volume snapshots %s/%s (%s)", t.PoolName, t.VolumeName, t.ProjectName)
}

func (t VolumePruneTask) Execute(x *ExecCtx) error {
	ctx := x.Ctx
	logger := x.Logger.With("project", t.ProjectName, "pool", t.PoolName, "volume", t.VolumeName)

	now := time.Now()

	// Source prune (direkt auf Incus Source)
	srcClient := x.Source.Server(t.ProjectName)
	if err := pruneSourceVolume(logger, "source", srcClient, t.PoolName, t.VolumeName, t.SourcePolicy, now, x.DryRunPrune); err != nil {
		return fmt.Errorf("source prune failed: %w", err)
	}

	// Target prune (über Target-Interface)
	subject := fmt.Sprintf("%s/%s", t.PoolName, t.VolumeName)
	if err := pruneTargetWithCtx(ctx, logger, "target", x.Target, transfer.KindVolume, subject, t.TargetPolicy, now, x.DryRunPrune); err != nil {
		return fmt.Errorf("target prune failed: %w", err)
	}

	return nil
}

func volumeKey(project, pool, volume string) string {
	return fmt.Sprintf("%s/%s/%s", project, pool, volume)
}
