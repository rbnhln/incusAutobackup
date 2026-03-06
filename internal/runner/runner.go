package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/rbnhln/incusAutobackup/internal/source"
	"github.com/rbnhln/incusAutobackup/internal/target"
	"github.com/rbnhln/incusAutobackup/internal/transfer"
)

type ExecCtx struct {
	Ctx         context.Context
	Logger      *slog.Logger
	Source      source.Source
	Target      target.Target
	DryRunCopy  bool
	DryRunPrune bool

	// Stores results of snapshot preparation phase
	VolumeSnapshots   map[string]transfer.Artifact
	InstanceSnapshots map[string]transfer.Artifact
}

type Task interface {
	Name() string
	Execute(x *ExecCtx) error
}

type Plan struct {
	Tasks []Task
}

func (p *Plan) Add(t Task) {
	p.Tasks = append(p.Tasks, t)
}

func (p *Plan) Execute(x *ExecCtx) error {
	total := len(p.Tasks)
	failed := 0
	var errs []error

	for i, task := range p.Tasks {
		x.Logger.Info("executing task", "i", i+1, "n", total, "task", task.Name())

		err := task.Execute(x)
		if err != nil {
			failed++
			x.Logger.Error("task failed", "task", task.Name(), "error", err, "failed", failed, "total", total)
			errs = append(errs, fmt.Errorf("task %q failed: %w", task.Name(), err))
			continue
		}
	}

	if failed > 0 {
		x.Logger.Error("plan finished with errors", "failed", failed, "total", total)
		return errors.Join(errs...)
	}

	x.Logger.Info("plan finished successfully", "total", total)
	return nil
}
