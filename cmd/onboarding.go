package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	incus "github.com/lxc/incus/v6/client"
	"github.com/rbnhln/incusAutobackup/internal/config"
)

func Onboard(opts OnboardOptions) error {
	err := onboardHost("source", opts.SourceURL, opts.SourceToken, opts)
	if err != nil {
		return err
	}

	err = onboardHost("target", opts.TargetURL, opts.TargetToken, opts)
	if err != nil {
		return err
	}

	return nil
}

func onboardHost(role, hostURL, token string, opts OnboardOptions) error {
	if hostURL == "" || token == "" {
		return fmt.Errorf("missing %s URL or token", role)
	}

	// Step 1 Get Server cert
	serverPEM, err := config.FetchServerCertAndVerifiy(hostURL, token)
	if err != nil {
		return err
	}

	// Step 2 Save servercert
	//serverCertPath := serverCertPath(opts.IABCredDIr, hostURL)
	serverCertPath, err := config.ServerCertPath(opts.IABCredDIr, hostURL)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Dir(serverCertPath), 0o700)
	if err != nil {
		return fmt.Errorf("failed to create server cert directory: %w", err)
	}
	err = os.WriteFile(serverCertPath, []byte(serverPEM), 0o600)
	if err != nil {
		return fmt.Errorf("failed to save %s server cert: %w", role, err)
	}

	// Step 3 Create or load Cert/Key pair for IAB
	clientCert, clientKey, err := config.GenerateOrLoadClientCreds(opts.IABCredDIr)
	if err != nil {
		return err
	}

	// Step 3.5 Test if cert is already trusted
	trusted, err := isTrusted(serverPEM, hostURL, role, clientCert, clientKey)
	if err != nil {
		return err
	}
	if trusted {
		fmt.Printf("[%s] client cert already trusted\n", role)
		return nil
	}

	// Step 4 Trust-token registration
	err = config.TrustClientCertWithToken(hostURL, token, string(clientCert), string(clientKey), opts.UUID)
	if err != nil {
		return err
	}

	// Step 5 Test connection
	trusted, err = isTrusted(serverPEM, hostURL, role, clientCert, clientKey)
	if err != nil {
		return err
	}
	if !trusted {
		return fmt.Errorf("[%s] trust call succeeded but connection is still not trusted", role)
	}
	fmt.Printf("[%s] Onboarding OK\n", role)
	return nil
}

// func serverCertPath(baseDir, hostURL string) string {
// 	u, _ := url.Parse(hostURL)
// 	host := strings.ReplaceAll(u.Hostname(), ":", "_")
// 	return filepath.Join(baseDir, "servers", host+".pem")
// }

func isTrusted(serverPEM, hostURL, role string, clientCert, clientKey []byte) (bool, error) {
	args := &incus.ConnectionArgs{
		TLSServerCert:      serverPEM,
		TLSClientCert:      string(clientCert),
		TLSClientKey:       string(clientKey),
		InsecureSkipVerify: false,
	}

	client, err := incus.ConnectIncus(hostURL, args)
	if err != nil {
		return false, fmt.Errorf("connection failf for %s: %w", role, err)
	}

	sInfo, _, err := client.GetServer()
	if err != nil {
		return false, fmt.Errorf("connection test failed for %s: %w", role, err)
	}
	if strings.ToLower(sInfo.Auth) != "trusted" {
		return false, nil
	}

	fmt.Printf("Connected to %s server (%s) with version: %s \n", role, sInfo.Environment.ServerName, sInfo.Environment.ServerVersion)
	return true, nil
}
