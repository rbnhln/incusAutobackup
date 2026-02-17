package main

import (
	"fmt"
	"os"

	incus "github.com/lxc/incus/v6/client"
	"github.com/rbnhln/incusAutobackup/internal/config"
)

func (app *application) GetHostByRole(role string) (config.Host, error) {
	for _, h := range app.config.Hosts {
		if h.Role == role {
			return h, nil
		}
	}

	return config.Host{}, fmt.Errorf("no host with role '%s' found in config", role)
}

func (app *application) ConnectToHost(host config.Host) (incus.InstanceServer, error) {
	iabDir := app.config.IAB.IABCredDir
	if iabDir == "" {
		return nil, fmt.Errorf("config.iab.iabCredDir is empty")
	}

	certPath := config.ClientCertPath(iabDir)
	keyPath := config.ClientKeyPath(iabDir)
	serverCertPath, err := config.ServerCertPath(iabDir, host.URL)
	if err != nil {
		return nil, err
	}

	cert, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cert file (%s): %w", certPath, err)
	}

	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file (%s): %w", keyPath, err)
	}

	serverCert, err := os.ReadFile(serverCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read server cert file (%s): %w", serverCertPath, err)
	}

	args := &incus.ConnectionArgs{
		TLSServerCert:      string(serverCert),
		TLSClientCert:      string(cert),
		TLSClientKey:       string(key),
		InsecureSkipVerify: false,
	}

	client, err := incus.ConnectIncus(host.URL, args)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to incus host %s (%s): %w", host.Name, host.URL, err)
	}

	return client, nil
}
