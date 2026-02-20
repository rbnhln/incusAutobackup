package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	incus "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
)

type ExecCtx struct {
	Ctx               context.Context
	Logger            *slog.Logger
	Source            incus.InstanceServer
	Target            incus.InstanceServer
	DryRunCopy        bool
	DryRunPrune       bool
	StopInstances     bool
	VolumeSnapshots   map[string]*api.StorageVolume
	InstanceSnapshots map[string]*api.Instance
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
