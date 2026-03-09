package main

import (
	"fmt"

	incus "github.com/lxc/incus/v6/client"
	"github.com/rbnhln/incusAutobackup/internal/config"
	tgtinterface "github.com/rbnhln/incusAutobackup/internal/target"
	tgtincus "github.com/rbnhln/incusAutobackup/internal/target/incus"
)

func (app *application) NewTarget(host config.TargetHost, client incus.InstanceServer) (tgtinterface.Target, error) {
	switch host.Type {
	case config.TargetTypeIncus:
		return tgtincus.New(client, tgtincus.Options{
			Name: host.Name,
		})
	default:
		return nil, fmt.Errorf("unsupported target type %q for target %q", host.Type, host.Name)
	}
}
