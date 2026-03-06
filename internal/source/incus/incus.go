package incus

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	incus "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
	"github.com/rbnhln/incusAutobackup/internal/transfer"
)

type Options struct {
	ProjectName   string
	StopIfRunning bool
}

type Source struct {
	logger *slog.Logger
	server incus.InstanceServer
	opts   Options
}

func New(logger *slog.Logger, server incus.InstanceServer, opts Options) *Source {
	return &Source{logger: logger, server: server, opts: opts}
}

func (s *Source) Server(projectName string) incus.InstanceServer {
	if projectName == "" {
		projectName = s.opts.ProjectName
	}
	return s.server.UseProject(projectName)
}

func (s *Source) snapshotName(now time.Time) string {
	return fmt.Sprintf("IAB_%s", now.Format("20060102-150405"))
}

func (s *Source) PrepareInstance(ctx context.Context, projectName, instanceName string) (transfer.Artifact, error) {
	_ = ctx // incus operations currently not ctx-aware

	if projectName == "" {
		projectName = s.opts.ProjectName
	}

	logger := s.logger.With("project", projectName, "instance", instanceName)
	server := s.Server(projectName)

	// 1 Check if instance exists
	_, _, err := server.GetInstance(instanceName)
	if err != nil {
		return transfer.Artifact{}, fmt.Errorf("get source instance %s failed: %w", instanceName, err)
	}

	// 2 Optional: stop if running
	wasRunning := false
	if s.opts.StopIfRunning {
		state, _, err := server.GetInstanceState(instanceName)
		if err != nil {
			return transfer.Artifact{}, fmt.Errorf("get instance state %s failed: %w", instanceName, err)
		}
		if state != nil && state.Status == "Running" {
			wasRunning = true
			logger.Info("stopping instance for snapshot/copy")

			op, err := server.UpdateInstanceState(instanceName, api.InstanceStatePut{
				Action:  "stop",
				Timeout: 300,
				Force:   false,
			}, "")
			if err != nil {
				return transfer.Artifact{}, fmt.Errorf("stop instance %q failed: %w", instanceName, err)
			}
			if err := op.Wait(); err != nil {
				return transfer.Artifact{}, fmt.Errorf("stop instance %q operation failed: %w", instanceName, err)
			}
		}
	}

	if s.opts.StopIfRunning && wasRunning {
		defer func() {
			logger.Info("starting instance after snapshot")
			op, err := server.UpdateInstanceState(instanceName, api.InstanceStatePut{
				Action:  "start",
				Timeout: 300,
			}, "")
			if err != nil {
				logger.Error("failed to start instance after snapshot", "error", err)
				return
			}
			if err := op.Wait(); err != nil {
				logger.Error("start instance operation failed after snapshot", "error", err)
			}
		}()
	}

	// 3 Create Snapshot
	now := time.Now()
	snapName := s.snapshotName(now)
	logger.Info("creating instance snapshot", "snapshot", snapName)

	opSnap, err := server.CreateInstanceSnapshot(instanceName, api.InstanceSnapshotsPost{
		Name:     snapName,
		Stateful: false,
	})
	if err != nil {
		return transfer.Artifact{}, fmt.Errorf("create snapshot for instance %s failed: %w", instanceName, err)
	}
	if err := opSnap.Wait(); err != nil {
		return transfer.Artifact{}, fmt.Errorf("create snapshot for instance %s operation failed: %w", instanceName, err)
	}

	point := transfer.RecoveryPoint{
		Kind:      transfer.KindInstance,
		Project:   projectName,
		Subject:   instanceName,
		Name:      snapName,
		CreatedAt: now,
	}

	arti := transfer.Artifact{
		Point: point,
		Open: func(ctx context.Context) (io.ReadCloser, error) {
			_ = ctx
			return nil, transfer.ErrNotSupported
		},
		Size: -1,
	}

	return arti, nil
}

func (s *Source) PrepareVolume(ctx context.Context, projectName, poolName, volumeName string) (transfer.Artifact, error) {
	_ = ctx

	if projectName == "" {
		projectName = s.opts.ProjectName
	}

	logger := s.logger.With("project", projectName, "pool", poolName, "volume", volumeName)
	server := s.Server(projectName)

	// 1 Check if volume exists on source pool
	_, _, err := server.GetStoragePoolVolume(poolName, "custom", volumeName)
	if err != nil {
		return transfer.Artifact{}, fmt.Errorf("volume not found on source: %w", err)
	}

	// 2 Create snapshot
	now := time.Now()
	snapName := s.snapshotName(now)
	logger.Info("creating volume snapshot", "snapshot", snapName)

	req := api.StorageVolumeSnapshotsPost{Name: snapName}
	op, err := server.CreateStoragePoolVolumeSnapshot(poolName, "custom", volumeName, req)
	if err != nil {
		return transfer.Artifact{}, fmt.Errorf("failed to create snapshot: %w", err)
	}
	if err := op.Wait(); err != nil {
		return transfer.Artifact{}, fmt.Errorf("snapshot operation failed: %w", err)
	}

	point := transfer.RecoveryPoint{
		Kind:      transfer.KindVolume,
		Project:   projectName,
		Subject:   fmt.Sprintf("%s/%s", poolName, volumeName),
		Name:      snapName,
		CreatedAt: now,
	}

	arti := transfer.Artifact{
		Point: point,
		Open: func(ctx context.Context) (io.ReadCloser, error) {
			_ = ctx
			return nil, transfer.ErrNotSupported
		},
		Size: -1,
	}
	return arti, nil
}
