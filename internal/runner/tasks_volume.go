package runner

import (
	"fmt"
	"time"

	"github.com/rbnhln/incusAutobackup/internal/backup"
)

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

func (t VolumeCopyTask) Name() string {
	return fmt.Sprintf("copy volume %s/%s (%s)", t.PoolName, t.VolumeName, t.ProjectName)
}

func (t VolumeCopyTask) Execute(x *ExecCtx) error {
	logger := x.Logger.With("project", t.ProjectName, "pool", t.PoolName, "volume", t.VolumeName)

	if x.DryRunCopy {
		logger.Info("dry-run copy: skipping snapshot and copy process")
		return nil
	}

	source := x.Source.UseProject(t.ProjectName)
	target := x.Target.UseProject(t.ProjectName)

	return backup.SyncVolume(logger, source, target, t.PoolName, t.VolumeName, t.Mode)
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
