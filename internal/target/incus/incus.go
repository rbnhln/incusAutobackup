package incus

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	incus "github.com/lxc/incus/v6/client"
	"github.com/rbnhln/incusAutobackup/internal/target"
	"github.com/rbnhln/incusAutobackup/internal/transfer"
)

type Options struct {
	ProjectName string
	Name        string
}

type Target struct {
	server incus.InstanceServer
	opts   Options
}

func New(server incus.InstanceServer, opts Options) (*Target, error) {
	if strings.TrimSpace(opts.ProjectName) == "" {
		return nil, fmt.Errorf("incus target requires project name")
	}
	return &Target{server: server, opts: opts}, nil
}

func (t *Target) Name() string {
	return fmt.Sprintf("incus target %s (project: %s)", t.opts.Name, t.opts.ProjectName)
}

func (t *Target) scoped() incus.InstanceServer {
	return t.server.UseProject(t.opts.ProjectName)
}

func (t *Target) Ping(ctx context.Context) error {
	_ = ctx
	_, _, err := t.scoped().GetServer()
	return err
}

func (t *Target) Close() error { return nil }

func (t *Target) Put(ctx context.Context, logger *slog.Logger, arti transfer.Artifact) error {
	_ = ctx
	_ = logger
	_ = arti
	return transfer.ErrNotSupported
}

func (t *Target) IncusCopy(ctx context.Context, logger *slog.Logger, src incus.InstanceServer, point transfer.RecoveryPoint, opts target.IncusCopyOptions) error {
	_ = ctx

	switch point.Kind {
	case transfer.KindInstance:
		inst, _, err := src.GetInstance(point.Subject)
		if err != nil {
			return fmt.Errorf("get source instance %q failed: %w", point.Subject, err)
		}
		// 4.1 Need to change the storage pool, did not find Pool flag in copy args
		// 4.2 filter devices which are not present on the target host
		instCopy := *inst
		instCopy.Devices = cloneDevices(inst.Devices)

		// root disk change
		if opts.TargetPool != "" {
			applyTargetPoolToRootDisk(instCopy.Devices, opts.TargetPool)
		}

		// sanitze devices for target host, drop with warn if not present
		err = sanitizeDevicesForTarget(logger, t.scoped(), instCopy.Devices, opts.ExcludeDevices)
		if err != nil {
			return fmt.Errorf("sanitize devices failed: %w", err)
		}

		copyArgs := incus.InstanceCopyArgs{
			Name:                point.Subject,
			Mode:                opts.Mode,
			InstanceOnly:        false,
			Refresh:             true,
			Live:                false,
			RefreshExcludeOlder: false,
			AllowInconsistent:   false,
		}

		// perform copy
		opCopy, err := t.scoped().CopyInstance(src, instCopy, &copyArgs)
		if err != nil {
			return fmt.Errorf("copy instance %s to target failed: %w", point.Subject, err)
		}
		err = opCopy.Wait()
		if err != nil {
			return fmt.Errorf("copy instance %s operation failed: %w", point.Subject, err)
		}

		logger.Info("instance sync successful")
		return nil

	case transfer.KindVolume:
		poolName, volName, err := splitPoolVolume(point.Subject)
		if err != nil {
			return err
		}

		// get incus volume form source server
		incusVolume, _, err := src.GetStoragePoolVolume(poolName, "custom", volName)
		if err != nil {
			return fmt.Errorf("get source volume %q failed: %w", point.Subject, err)
		}

		copyArgs := incus.StoragePoolVolumeCopyArgs{
			Name:       volName,
			Mode:       opts.Mode,
			VolumeOnly: false,
			Refresh:    true,
		}

		// perform copy
		opCopy, err := t.scoped().CopyStoragePoolVolume(poolName, src, poolName, *incusVolume, &copyArgs)
		if err != nil {
			return fmt.Errorf("failed to start copy operation: %w", err)
		}

		err = opCopy.Wait()
		if err != nil {
			return fmt.Errorf("copy operation failed: %w", err)
		}

		logger.Info("Volume sync successful")
		return nil
	default:
		return fmt.Errorf("unsupported kind: %s", point.Kind)
	}
}

func (t *Target) List(ctx context.Context, kind transfer.Kind, subject string) ([]transfer.RecoveryPoint, error) {
	_ = ctx
	switch kind {
	case transfer.KindInstance:
		snaps, err := t.scoped().GetInstanceSnapshots(subject)
		if err != nil {
			return nil, err
		}
		out := make([]transfer.RecoveryPoint, 0, len(snaps))
		for _, s := range snaps {
			out = append(out, transfer.RecoveryPoint{
				Kind:      transfer.KindInstance,
				Project:   t.opts.ProjectName,
				Subject:   subject,
				Name:      stripAfterLastSlash(s.Name),
				CreatedAt: s.CreatedAt,
			})
		}
		return out, nil
	case transfer.KindVolume:
		poolName, volName, err := splitPoolVolume(subject)
		if err != nil {
			return nil, err
		}

		snaps, err := t.scoped().GetStoragePoolVolumeSnapshots(poolName, "custom", volName)
		if err != nil {
			return nil, err
		}

		out := make([]transfer.RecoveryPoint, 0, len(snaps))
		for _, s := range snaps {
			out = append(out, transfer.RecoveryPoint{
				Kind:      transfer.KindVolume,
				Project:   t.opts.ProjectName,
				Subject:   subject,
				Name:      stripAfterLastSlash(s.Name),
				CreatedAt: s.CreatedAt,
			})
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported king: %s", kind)
	}
}

func (t *Target) Delete(ctx context.Context, point transfer.RecoveryPoint) error {
	_ = ctx

	switch point.Kind {
	case transfer.KindInstance:
		opDel, err := t.scoped().DeleteInstanceSnapshot(point.Subject, point.Name)
		if err != nil {
			return err
		}
		return opDel.Wait()

	case transfer.KindVolume:
		poolName, volName, err := splitPoolVolume(point.Subject)
		if err != nil {
			return err
		}
		opDel, err := t.scoped().DeleteStoragePoolVolumeSnapshot(poolName, "custom", volName, point.Name)
		if err != nil {
			return err
		}
		return opDel.Wait()

	default:
		return fmt.Errorf("unsupported kind: %s", point.Kind)
	}

}
