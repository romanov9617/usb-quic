package usbip

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/quic-go/quic-go"
)

const quicALPN = "usb-quic"

const (
	privateKeyBits      = 2048
	serialNumberBits    = 128
	certificateLifetime = 24 * time.Hour
)

// QUICClientTLS creates TLS settings for the QUIC client proxy.
func QUICClientTLS() *tls.Config {
	//nolint:exhaustruct // Only QUIC ALPN and development certificate verification policy differ from defaults.
	return &tls.Config{
		NextProtos:         []string{quicALPN},
		InsecureSkipVerify: true, //nolint:gosec // Local development proxy uses ephemeral self-signed QUIC certificates.
	}
}

// QUICServerTLS creates TLS settings for the QUIC server proxy.
func QUICServerTLS() (*tls.Config, error) {
	key, err := rsa.GenerateKey(rand.Reader, privateKeyBits)
	if err != nil {
		return nil, fmt.Errorf("generate tls key: %w", err)
	}

	certificate, err := newSelfSignedCertificate(key)
	if err != nil {
		return nil, err
	}

	//nolint:exhaustruct // Only certificate and QUIC ALPN differ from TLS defaults.
	return &tls.Config{
		Certificates: []tls.Certificate{certificate},
		NextProtos:   []string{quicALPN},
	}, nil
}

// QUICConfig creates QUIC transport settings for byte-stream proxying.
func QUICConfig() *quic.Config {
	//nolint:exhaustruct // Only datagram support is explicitly controlled for byte-stream proxying.
	return &quic.Config{
		EnableDatagrams: false,
	}
}

func newSelfSignedCertificate(key *rsa.PrivateKey) (tls.Certificate, error) {
	serialNumber, err := newSerialNumber()
	if err != nil {
		return tls.Certificate{}, err
	}

	now := time.Now()
	//nolint:exhaustruct // Self-signed development certificate only needs fields used by TLS handshake.
	template := x509.Certificate{
		SerialNumber: serialNumber,
		//nolint:exhaustruct // CommonName is sufficient for this self-signed development certificate.
		Subject: pkix.Name{
			CommonName: "usb-quic",
		},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(certificateLifetime),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create tls certificate: %w", err)
	}

	return encodeKeyPair(derBytes, key)
}

func newSerialNumber() (*big.Int, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), serialNumberBits)

	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("generate tls serial number: %w", err)
	}

	return serialNumber, nil
}

func encodeKeyPair(derBytes []byte, key *rsa.PrivateKey) (tls.Certificate, error) {
	//nolint:exhaustruct // PEM certificate block does not require headers.
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})
	//nolint:exhaustruct // PEM key block does not require headers.
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	certificate, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("load tls key pair: %w", err)
	}

	return certificate, nil
}
