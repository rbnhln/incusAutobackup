package pipeline

import (
	"context"
	"log/slog"

	"github.com/rbnhln/incusAutobackup/internal/source"
	"github.com/rbnhln/incusAutobackup/internal/target"
	"github.com/rbnhln/incusAutobackup/internal/transfer"
)

func Store(ctx context.Context, logger *slog.Logger, src source.Source, dst target.Target, arti transfer.Artifact, incusOpts target.IncusCopyOptions) error {
	// check if target is native incus(os) system

	t, ok := dst.(target.NativeIncusTarget)
	// if true perform incus copy
	if ok {
		srcServer := src.Server(arti.Point.Project)
		return t.IncusCopy(ctx, logger, srcServer, arti.Point, incusOpts)
	}

	// else use generic Put function
	return dst.Put(ctx, logger, arti)
}
