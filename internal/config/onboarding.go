package config

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	cryptotls "crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lxc/incus/v6/shared/api"
	incustls "github.com/lxc/incus/v6/shared/tls"
)

// Token Parsing
func parseToken(token string) (*api.CertificateAddToken, error) {
	return incustls.CertificateTokenDecode(token)
}

// Collect server Certificate by URL
func FetchServerCertAndVerifiy(hostURL string, token string) (serverPEM string, err error) {
	tokenData, err := parseToken(token)
	if err != nil {
		return "", fmt.Errorf("bad token: %w", err)
	}

	cert, err := incustls.GetRemoteCertificate(hostURL+"/1.0", "")
	if err != nil {
		return "", fmt.Errorf("failed to get cert from TLS request, make sure you use https: %w", err)

	}

	gotFP := incustls.CertFingerprint(cert)
	if gotFP != tokenData.Fingerprint {
		return "", fmt.Errorf("fingerprint mismatch between TLS-Cert and Token")
	}

	serverPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}))
	return serverPEM, nil

}

// Load or generate new client cert/key pair
func GenerateOrLoadClientCreds(dir string) (certPEM, keyPEM []byte, err error) {
	certPath := filepath.Join(dir, "iab_client.crt")
	keyPath := filepath.Join(dir, "iab_client.key")

	// load if certs files are available
	if cert, err := os.ReadFile(certPath); err == nil {
		if key, err2 := os.ReadFile(keyPath); err2 == nil {
			return cert, key, nil
		}
	}

	err = os.MkdirAll(dir, 0o700)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create cert folder, %s", dir)
	}

	// Create new key
	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create new private key: %w", err)
	}

	// Create self-signed client cert
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "incusAutobackup"},
		NotBefore:    time.Now().Add(-5 * time.Minute),
		NotAfter:     time.Now().AddDate(5, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create new certificate: %w", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	// save the cert
	err = os.WriteFile(certPath, certPEM, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("failed tp save cert: %w", err)
	}
	err = os.WriteFile(keyPath, keyPEM, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to save key: %w", err)
	}

	return certPEM, keyPEM, nil

}

func TrustClientCertWithToken(hostURL, token, clientCertPEM, clientKeyPEM, UUID string) error {
	tokenData, err := parseToken(token)
	if err != nil {
		return fmt.Errorf("bad token: %w", err)
	}

	keyPair, err := cryptotls.X509KeyPair([]byte(clientCertPEM), []byte(clientKeyPEM))
	if err != nil {
		return fmt.Errorf("invalid client cert/key pair: %w", err)
	}

	payload := api.CertificatesPost{
		CertificatePut: api.CertificatePut{
			Name:        tokenData.ClientName,
			Type:        api.CertificateTypeClient,
			Certificate: clientCertPEM,
			Description: "incusAutoback uuid=" + UUID,
		},
		TrustToken: token,
	}

	body, _ := json.Marshal(payload)

	tlsCfg := &cryptotls.Config{
		InsecureSkipVerify: true,
		Certificates:       []cryptotls.Certificate{keyPair},
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("no peer cert")
			}
			sum := sha256.Sum256(rawCerts[0])
			gotFP := fmt.Sprintf("%x", sum)
			if gotFP != tokenData.Fingerprint {
				return fmt.Errorf("server fingerprint mismatch: got %s want %s", gotFP, tokenData.Fingerprint)
			}
			return nil
		},
	}

	httpClient := &http.Client{
		Transport: &http.Transport{TLSClientConfig: tlsCfg},
	}

	req, _ := http.NewRequest("POST", hostURL+"/1.0/certificates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("trust POST failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("trust failed: status %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return nil
}
