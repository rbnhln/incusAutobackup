package runner

import (
	"fmt"
	"time"

	"github.com/rbnhln/incusAutobackup/internal/backup"
)

type InstanceCopyTask struct {
	ProjectName    string
	InstanceName   string
	Mode           string
	PoolName       string
	ExcludeDevices []string
}

type InstancePruneTask struct {
	ProjectName  string
	InstanceName string
	SourcePolicy string
	TargetPolicy string
}

func (t InstanceCopyTask) Name() string {
	return fmt.Sprintf("copy instance %s (%s)", t.InstanceName, t.ProjectName)
}

func (t InstanceCopyTask) Execute(x *ExecCtx) error {
	logger := x.Logger.With("project", t.ProjectName, "instance", t.InstanceName)

	if x.DryRunCopy {
		logger.Info("dry-run copy: skipping snapshot and copy process")
		return nil
	}

	source := x.Source.UseProject(t.ProjectName)
	target := x.Target.UseProject(t.ProjectName)

	return backup.SyncInstance(logger, source, target, t.InstanceName, t.Mode, t.PoolName, x.StopInstances, t.ExcludeDevices)
}

func (t InstancePruneTask) Name() string {
	return fmt.Sprintf("prune instance snapshot %s (%s)", t.InstanceName, t.ProjectName)
}

func (t InstancePruneTask) Execute(x *ExecCtx) error {
	logger := x.Logger.With("project", t.ProjectName, "instance", t.InstanceName)

	source := x.Source.UseProject(t.ProjectName)
	target := x.Target.UseProject(t.ProjectName)

	return backup.PruneInstance(logger, source, target, t.InstanceName, t.SourcePolicy, t.TargetPolicy, time.Now(), x.DryRunPrune)
}
