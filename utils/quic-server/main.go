// Command quic-echo-server runs a minimal QUIC echo server over UDP.
//
// The server listens on a given address, accepts QUIC connections and streams,
// and echoes stream payload back to the sender. It uses a self-signed
// certificate generated at startup and logs events via slog.
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"os"
	"sync/atomic"
	"time"

	quic "github.com/quic-go/quic-go"
)

// alpn is the Application-Layer Protocol Negotiation identifier used by this server.
const alpn = "quic-echo"

// server holds the QUIC listener and counters used for structured logging.
type server struct {
	logger    *slog.Logger
	listener  *quic.Listener
	connSeq   atomic.Uint64
	streamSeq atomic.Uint64
}

// main configures structured logging and runs the server.
// It exits with a non-zero status on fatal errors.
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	if err := run(context.Background(), logger); err != nil {
		// Fatal only here: keep helpers testable and error-returning.
		logger.Error("fatal", "err", err)
		os.Exit(1)
	}
}

// run prepares TLS and QUIC listener configuration and starts serving.
func run(ctx context.Context, logger *slog.Logger) error {
	addr := "0.0.0.0:4242"

	tlsConf, err := buildTLSConfig(logger)
	if err != nil {
		return fmt.Errorf("build tls config: %w", err)
	}

	ln, err := quic.ListenAddr(addr, tlsConf, &quic.Config{})
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	s := &server{
		logger:   logger.With("component", "server", "addr", addr, "proto", "udp"),
		listener: ln,
	}

	s.logger.Info("started")
	return s.serve(ctx)
}

// serve accepts incoming QUIC connections until ctx is canceled or an error occurs.
func (s *server) serve(ctx context.Context) error {
	for {
		conn, err := s.listener.Accept(ctx)
		if err != nil {
			// Context cancellation is a graceful shutdown path.
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				s.logger.Info("accept loop stopped by context", "err", err)
				return nil
			}
			return fmt.Errorf("accept conn: %w", err)
		}

		connID := s.connSeq.Add(1)
		l := s.logger.With(
			"component", "conn",
			"conn_id", connID,
			"remote", conn.RemoteAddr().String(),
		)

		l.Info("accepted")
		go func() {
			if err := s.handleConn(ctx, conn, connID, l); err != nil {
				l.Warn("connection handler ended with error", "err", err)
			}
		}()
	}
}

// handleConn accepts streams from conn and starts an echo handler for each stream.
func (s *server) handleConn(ctx context.Context, conn *quic.Conn, _ uint64, l *slog.Logger) error {
	defer func() {
		l.Info("closing")
		_ = conn.CloseWithError(0, "server closing")
	}()

	for {
		st, err := conn.AcceptStream(ctx)
		if err != nil {
			// Client close or context cancellation commonly ends the stream loop.
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				l.Info("accept stream stopped by context", "err", err)
				return nil
			}
			return fmt.Errorf("accept stream: %w", err)
		}

		streamID := s.streamSeq.Add(1)
		sl := l.With("component", "stream", "stream_id", streamID)

		sl.Debug("opened")
		go func() {
			if err := echoStream(st, sl); err != nil {
				sl.Warn("echo ended with error", "err", err)
			}
		}()
	}
}

// echoStream reads from st and writes back to st until EOF or an error occurs.
func echoStream(st *quic.Stream, l *slog.Logger) error {
	defer func() {
		_ = st.Close()
		l.Debug("closed")
	}()

	start := time.Now()
	// io.Copy reads from the stream and writes back to the same stream (echo).
	n, err := io.Copy(st, st)
	dur := time.Since(start)

	// io.EOF is expected when the peer closes its write side.
	if err != nil && !errors.Is(err, io.EOF) {
		l.Warn("copy failed", "bytes", n, "dur", dur, "err", err)
		return fmt.Errorf("io.Copy: %w", err)
	}

	l.Info("echo done", "bytes", n, "dur", dur)
	return nil
}

// buildTLSConfig returns a TLS configuration with a freshly generated self-signed certificate.
// The certificate is suitable for local development and advertises the [alpn] protocol.
func buildTLSConfig(l *slog.Logger) (*tls.Config, error) {
	l = l.With("component", "tls")
	l.Debug("generating self-signed certificate")

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("ed25519 keygen: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("serial: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serial,
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),

		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		// DNSNames is set for "localhost" to support local testing.
		DNSNames: []string{"localhost"},
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, priv.Public(), priv)
	if err != nil {
		return nil, fmt.Errorf("create cert: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	keyBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("marshal private key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse keypair: %w", err)
	}

	l.Info("certificate ready", "alpn", alpn)

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{alpn},
	}, nil
}
