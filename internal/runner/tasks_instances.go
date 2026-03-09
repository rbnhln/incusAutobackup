package runner

import (
	"fmt"
	"time"

	"github.com/rbnhln/incusAutobackup/internal/pipeline"
	"github.com/rbnhln/incusAutobackup/internal/target"
	"github.com/rbnhln/incusAutobackup/internal/transfer"
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

type InstanceSourcePruneTask struct {
	ProjectName  string
	InstanceName string
	SourcePolicy string
}

type InstanceTargetPruneTask struct {
	ProjectName  string
	InstanceName string
	TargetPolicy string
}

func (t InstanceSnapshotTask) Name() string {
	return fmt.Sprintf("snapshot instance %s (%s)", t.InstanceName, t.ProjectName)
}

func (t InstanceSnapshotTask) Execute(x *ExecCtx) error {
	ctx := x.Ctx
	logger := x.Logger.With("project", t.ProjectName, "instance", t.InstanceName)

	if x.DryRunCopy {
		logger.Info("dry-run: skipping snapshot")
		return nil
	}

	arti, err := x.Source.PrepareInstance(ctx, t.ProjectName, t.InstanceName)
	if err != nil {
		return err
	}

	key := instanceKey(t.ProjectName, t.InstanceName)
	x.InstanceSnapshots[key] = arti
	return nil
}

func (t InstanceCopyTask) Name() string {
	return fmt.Sprintf("copy instance %s (%s)", t.InstanceName, t.ProjectName)
}

func (t InstanceCopyTask) Execute(x *ExecCtx) error {
	ctx := x.Ctx
	logger := x.Logger.With("project", t.ProjectName, "instance", t.InstanceName)

	if x.DryRunCopy {
		logger.Info("dry-run: skipping copy")
		return nil
	}

	key := instanceKey(t.ProjectName, t.InstanceName)
	arti, ok := x.InstanceSnapshots[key]
	if !ok {
		logger.Warn("skipping copy: no snapshot was created (snapshot may have failed)")
		return fmt.Errorf("no snapshot result found for instance %s – snapshot phase likely failed", key)
	}

	incusOpts := target.IncusCopyOptions{
		Mode:           t.Mode,
		TargetPool:     t.PoolName,
		ExcludeDevices: t.ExcludeDevices,
	}

	return pipeline.Store(ctx, logger, x.Source, x.Target, arti, incusOpts)
}

func (t InstanceSourcePruneTask) Name() string {
	return fmt.Sprintf("prune source instance snapshots %s (%s)", t.InstanceName, t.ProjectName)
}

func (t InstanceSourcePruneTask) Execute(x *ExecCtx) error {
	logger := x.Logger.With("project", t.ProjectName, "instance", t.InstanceName)
	now := time.Now()

	srcClient := x.Source.Server(t.ProjectName)
	if err := pruneSourceInstance(logger, "source", srcClient, t.InstanceName, t.SourcePolicy, now, x.DryRunPrune); err != nil {
		return fmt.Errorf("source prune failed: %w", err)
	}
	return nil
}

func (t InstanceTargetPruneTask) Name() string {
	return fmt.Sprintf("prune target instance snapshots %s (%s)", t.InstanceName, t.ProjectName)
}

func (t InstanceTargetPruneTask) Execute(x *ExecCtx) error {
	ctx := x.Ctx
	logger := x.Logger.With("project", t.ProjectName, "instance", t.InstanceName)
	now := time.Now()

	if err := pruneTargetWithCtx(
		ctx, logger, "target", x.Target,
		transfer.KindInstance, t.InstanceName, t.TargetPolicy, t.ProjectName, now, x.DryRunPrune,
	); err != nil {
		return fmt.Errorf("target prune failed: %w", err)
	}
	return nil
}
