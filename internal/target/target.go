package target

import (
	"context"
	"log/slog"

	incus "github.com/lxc/incus/v6/client"
	"github.com/rbnhln/incusAutobackup/internal/transfer"
)

type IncusCopyOptions struct {
	Mode           string
	TargetPool     string
	ExcludeDevices []string
}

type Target interface {
	Name() string
	Ping(ctx context.Context) error
	Close() error

	// retention and prune
	List(ctx context.Context, kind transfer.Kind, subject string) ([]transfer.RecoveryPoint, error)
	Delete(ctx context.Context, point transfer.RecoveryPoint) error

	// export-based store (S3, SFT, fs, ...)
	Put(ctx context.Context, logger *slog.Logger, arti transfer.Artifact) error
}

type NativeIncusTarget interface {
	Target
	IncusCopy(ctx context.Context, logger *slog.Logger, src incus.InstanceServer, point transfer.RecoveryPoint, opts IncusCopyOptions) error
}
