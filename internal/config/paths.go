package config

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

func ClientCertPath(iabCredDir string) string {
	return filepath.Join(iabCredDir, "iab_client.crt")
}

func ClientKeyPath(iabCredDir string) string {
	return filepath.Join(iabCredDir, "iab_client.key")
}

func ServerCertPath(iabCredDir, hostURL string) (string, error) {
	u, err := url.Parse(hostURL)
	if err != nil {
		return "", fmt.Errorf("invalid host url %q: %w", hostURL, err)
	}
	host := strings.NewReplacer(":", "_", "/", "_").Replace(u.Hostname())
	return filepath.Join(iabCredDir, "servers", host+".pem"), nil
}
