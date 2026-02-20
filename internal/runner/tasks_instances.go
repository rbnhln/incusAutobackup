package runner

import (
	"fmt"
	"time"

	"github.com/rbnhln/incusAutobackup/internal/backup"
)

type InstanceSnapshotTask struct {
	ProjectName  string
	InstanceName string
}

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

func (t InstanceSnapshotTask) Name() string {
	return fmt.Sprintf("snapshot instance %s (%s)", t.InstanceName, t.ProjectName)
}

func (t InstanceSnapshotTask) Execute(x *ExecCtx) error {
	logger := x.Logger.With("project", t.ProjectName, "instance", t.InstanceName)

	if x.DryRunCopy {
		logger.Info("dry-run: skipping snapshot")
		return nil
	}

	source := x.Source.UseProject(t.ProjectName)

	inst, err := backup.SnapshotInstance(logger, source, t.InstanceName, x.StopInstances)
	if err != nil {
		return err
	}

	key := instanceKey(t.ProjectName, t.InstanceName)
	x.InstanceSnapshots[key] = inst
	return nil
}

func (t InstanceCopyTask) Name() string {
	return fmt.Sprintf("copy instance %s (%s)", t.InstanceName, t.ProjectName)
}

func (t InstanceCopyTask) Execute(x *ExecCtx) error {
	logger := x.Logger.With("project", t.ProjectName, "instance", t.InstanceName)

	if x.DryRunCopy {
		logger.Info("dry-run: skipping copy")
		return nil
	}

	key := instanceKey(t.ProjectName, t.InstanceName)
	inst, ok := x.InstanceSnapshots[key]
	if !ok {
		logger.Warn("skipping copy: no snapshot was created (snapshot may have failed)")
		return fmt.Errorf("no snapshot result found for instance %s – snapshot phase likely failed", key)
	}

	source := x.Source.UseProject(t.ProjectName)
	target := x.Target.UseProject(t.ProjectName)

	return backup.CopyInstance(logger, source, target, t.InstanceName, t.Mode, t.PoolName, t.ExcludeDevices, inst)
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

func instanceKey(project, instance string) string {
	return fmt.Sprintf("%s/%s", project, instance)
}
