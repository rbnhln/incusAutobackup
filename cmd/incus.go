package main

import (
	"fmt"
	"os"
	"strings"

	incus "github.com/lxc/incus/v6/client"
	"github.com/rbnhln/incusAutobackup/internal/config"
)

func (app *application) GetSourceHost() (config.SourceHost, error) {
	if strings.TrimSpace(app.config.Source.URL) == "" {
		return config.SourceHost{}, fmt.Errorf("source host URL is missing in config")
	}
	return app.config.Source, nil
}

func (app *application) GetTargets() ([]config.TargetHost, error) {
	if len(app.config.Targets) == 0 {
		return nil, fmt.Errorf("no targets configured")
	}
	return app.config.Targets, nil
}

func (app *application) ConnectToSource(host config.SourceHost) (incus.InstanceServer, error) {
	return app.connectIncusHost(host.Name, host.URL)
}

func (app *application) ConnectToTarget(host config.TargetHost) (incus.InstanceServer, error) {
	return app.connectIncusHost(host.Name, host.URL)
}

func (app *application) connectIncusHost(name, hostURL string) (incus.InstanceServer, error) {
	iabDir := app.config.IAB.IABCredDir
	if iabDir == "" {
		return nil, fmt.Errorf("config.iab.iabCredDir is empty")
	}

	certPath := config.ClientCertPath(iabDir)
	keyPath := config.ClientKeyPath(iabDir)
	serverCertPath, err := config.ServerCertPath(iabDir, hostURL)
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

	client, err := incus.ConnectIncus(hostURL, args)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to incus host %s (%s): %w", name, hostURL, err)
	}

	return client, nil
}
