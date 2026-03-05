package source

import (
	"context"

	incus "github.com/lxc/incus/v6/client"
	"github.com/rbnhln/incusAutobackup/internal/transfer"
)

type Source interface {
	PrepareInstance(ctx context.Context, instanceName string) (transfer.Artifact, error)
	PrepareVolume(ctx context.Context, poolName, volumeName string) (transfer.Artifact, error)
	Server(projectName string) incus.InstanceServer
}
