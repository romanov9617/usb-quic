package transport

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"testing"
	"time"

	"github.com/quic-go/quic-go"
)

const testQUICNextProto = "usb-quic-test"

func TestQUICStreamOpenerOpensBidirectionalStream(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	listener := listenQUIC(t)
	defer closeQUICListener(listener)

	serverStream, serverErrs := acceptQUICStream(ctx, listener)

	clientConn := dialQUIC(t, ctx, listener)
	defer closeQUICConn(clientConn)

	opener := NewQUICStreamOpener(clientConn)

	clientEndpoint, err := opener.OpenStream(ctx)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	writeString(t, clientEndpoint, "request")

	var accepted *quic.Stream

	select {
	case accepted = <-serverStream:
	case err = <-serverErrs:
		t.Fatalf("accept stream: %v", err)
	case <-ctx.Done():
		t.Fatalf("accept stream timed out: %v", ctx.Err())
	}

	if got := readString(t, accepted, len("request")); got != "request" {
		t.Fatalf("server read %q, want %q", got, "request")
	}

	writeString(t, accepted, "response")

	if got := readString(t, clientEndpoint, len("response")); got != "response" {
		t.Fatalf("client read %q, want %q", got, "response")
	}

	err = clientEndpoint.CloseWrite()
	if err != nil {
		t.Fatalf("close client write: %v", err)
	}

	if got := readEOF(t, accepted); got != "" {
		t.Fatalf("server read %q before EOF, want empty", got)
	}
}

func listenQUIC(t *testing.T) *quic.Listener {
	t.Helper()

	listener, err := quic.ListenAddr("127.0.0.1:0", testServerTLSConfig(t), nil)
	if err != nil {
		t.Fatalf("listen quic: %v", err)
	}

	return listener
}

func closeQUICListener(listener *quic.Listener) {
	_ = listener.Close()
}

func dialQUIC(t *testing.T, ctx context.Context, listener *quic.Listener) *quic.Conn {
	t.Helper()

	clientConn, err := quic.DialAddr(ctx, listener.Addr().String(), testClientTLSConfig(), nil)
	if err != nil {
		t.Fatalf("dial quic: %v", err)
	}

	return clientConn
}

func closeQUICConn(conn *quic.Conn) {
	_ = conn.CloseWithError(0, "")
}

func acceptQUICStream(
	ctx context.Context,
	listener *quic.Listener,
) (<-chan *quic.Stream, <-chan error) {
	serverStream := make(chan *quic.Stream, 1)
	serverErrs := make(chan error, 1)

	go func() {
		conn, err := listener.Accept(ctx)
		if err != nil {
			serverErrs <- err

			return
		}

		stream, err := conn.AcceptStream(ctx)
		if err != nil {
			serverErrs <- err

			return
		}

		serverStream <- stream
	}()

	return serverStream, serverErrs
}

func writeString(t *testing.T, writer io.Writer, value string) {
	t.Helper()

	_, err := io.WriteString(writer, value)
	if err != nil {
		t.Fatalf("write %q: %v", value, err)
	}
}

func readString(t *testing.T, reader io.Reader, size int) string {
	t.Helper()

	buffer := make([]byte, size)

	_, err := io.ReadFull(reader, buffer)
	if err != nil {
		t.Fatalf("read %d bytes: %v", size, err)
	}

	return string(buffer)
}

func readEOF(t *testing.T, reader io.Reader) string {
	t.Helper()

	buffer := make([]byte, 32)
	size, err := reader.Read(buffer)

	if err != io.EOF {
		t.Fatalf("read err=%v, want EOF", err)
	}

	return string(buffer[:size])
}

func testClientTLSConfig() *tls.Config {
	//nolint:exhaustruct // Test config intentionally sets only handshake-relevant fields.
	return &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // Test uses an ephemeral self-signed certificate.
		NextProtos:         []string{testQUICNextProto},
	}
}

func testServerTLSConfig(t *testing.T) *tls.Config {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	template := x509.Certificate{ //nolint:exhaustruct // Test certificate sets only required fields.
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{ //nolint:exhaustruct // Test subject only needs a common name.
			CommonName: testQUICNextProto,
		},
		NotBefore: time.Now().Add(-time.Minute),
		NotAfter:  time.Now().Add(time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature,
		DNSNames:  []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	keyDER := x509.MarshalPKCS1PrivateKey(key)
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:    "CERTIFICATE",
		Headers: map[string]string{},
		Bytes:   certDER,
	})
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: map[string]string{},
		Bytes:   keyDER,
	})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("parse key pair: %v", err)
	}

	//nolint:exhaustruct // Test config intentionally sets only handshake-relevant fields.
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{testQUICNextProto},
	}
}
