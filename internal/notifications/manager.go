package notifications

import (
	"context"
	"errors"
	"log/slog"

	"github.com/rbnhln/incusAutobackup/internal/config"
)

type Notifier interface {
	Name() string
	Start(ctx context.Context) error
	Finish(ctx context.Context, ok bool) error
}

type Manager struct {
	logger    *slog.Logger
	notifiers []Notifier
}

func NewManagerFromConfig(logger *slog.Logger, cfg config.Config) *Manager {
	m := &Manager{logger: logger}

	if cfg.IAB.HealthchecksURL != "" {
		m.notifiers = append(m.notifiers, NewHealthchecksNotifier(cfg.IAB.HealthchecksURL))
	}
	return m
}

func (m *Manager) Start(ctx context.Context) {
	for _, n := range m.notifiers {
		err := n.Start(ctx)
		if err != nil {
			m.logger.Warn("notification start failed", "notifier", n.Name(), "error", err)
		}
	}
}

func (m *Manager) Finish(ctx context.Context, ok bool) error {
	var errs []error
	for _, n := range m.notifiers {
		err := n.Finish(ctx, ok)
		if err != nil {
			m.logger.Warn("notification finish failed", "notifier", n.Name(), "error", err)
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
