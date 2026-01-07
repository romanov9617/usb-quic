// Command quic-echo-client runs an interactive QUIC echo client over UDP.
//
// The client connects to a QUIC echo server, opens a stream, and then sends
// user-provided lines and prints the echoed response. It supports basic
// commands to quit or open a new stream, and it stops gracefully on SIGINT/SIGTERM.
package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	quic "github.com/quic-go/quic-go"
)

// alpn is the Application-Layer Protocol Negotiation identifier required by the server.
const alpn = "quic-echo"

// config holds command-line configuration for the client.
type config struct {
	host string
	port int
}

// main parses flags, configures logging, and runs the interactive client.
// It exits with a non-zero status on fatal errors.
func main() {
	cfg := parseFlags()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	if err := run(context.Background(), logger, cfg); err != nil {
		logger.Error("fatal", "err", err)
		os.Exit(1)
	}
}

// parseFlags parses command-line flags and returns the resulting config.
func parseFlags() config {
	var cfg config

	flag.StringVar(&cfg.host, "host", "127.0.0.1", "QUIC server host or IP")
	flag.IntVar(&cfg.port, "port", 4242, "QUIC server UDP port")

	flag.Parse()
	return cfg
}

// run connects to the QUIC server and starts an interactive loop that
// sends lines and prints their echoed responses.
func run(ctx context.Context, logger *slog.Logger, cfg config) error {
	addr := fmt.Sprintf("%s:%d", cfg.host, cfg.port)

	ctx, cancel := withSignals(ctx, logger)
	defer cancel()

	logger.Info(
		"starting interactive quic echo client",
		"addr", addr,
	)

	tlsConf := &tls.Config{
		InsecureSkipVerify: true,           // Dev-only: accept self-signed certificates.
		NextProtos:         []string{alpn}, // Must match the server's ALPN.
	}

	conn, err := quic.DialAddr(ctx, addr, tlsConf, &quic.Config{
		KeepAlivePeriod: 10 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}
	defer func() { _ = conn.CloseWithError(0, "bye") }()

	logger.Info("connected", "remote", conn.RemoteAddr().String())

	st, err := conn.OpenStreamSync(ctx)
	if err != nil {
		return fmt.Errorf("open stream: %w", err)
	}
	defer func() { _ = st.Close() }()

	logger.Info(
		"stream opened",
		"commands", "/quit | /exit | /newstream",
	)

	// reader reads echoed data from the current stream.
	reader := bufio.NewReader(st)
	// input reads user input from stdin line-by-line.
	input := bufio.NewScanner(os.Stdin)

	for {
		select {
		case <-ctx.Done():
			logger.Info("stopping by context", "err", ctx.Err())
			return nil
		default:
		}

		fmt.Print("> ")
		if !input.Scan() {
			if err := input.Err(); err != nil {
				return fmt.Errorf("stdin scan: %w", err)
			}
			logger.Info("stdin closed")
			return nil
		}

		line := input.Text()
		cmd := strings.TrimSpace(line)

		switch cmd {
		case "/quit", "/exit":
			logger.Info("quit requested")
			return nil

		case "/newstream":
			// Open a fresh QUIC stream within the same connection.
			logger.Info("opening new stream")
			_ = st.Close()

			st, err = conn.OpenStreamSync(ctx)
			if err != nil {
				return fmt.Errorf("open new stream: %w", err)
			}
			reader = bufio.NewReader(st)
			logger.Info("new stream opened")
			continue
		}

		msg := line + "\n"

		start := time.Now()
		if _, err := io.WriteString(st, msg); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("write: %w", err)
		}

		// The echo server replies with the same bytes, line-terminated.
		echo, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
				logger.Info("stream closed by peer")
				return nil
			}
			return fmt.Errorf("read echo: %w", err)
		}

		rtt := time.Since(start)
		fmt.Printf("echo: %s", echo)

		logger.Debug(
			"roundtrip",
			"bytes", len(msg),
			"rtt", rtt,
		)
	}
}

// withSignals returns a child context that is canceled on SIGINT or SIGTERM.
// The returned cancel function should be called to release resources.
func withSignals(ctx context.Context, logger *slog.Logger) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	ch := make(chan os.Signal, 2)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-ch
		logger.Info("signal received, shutting down", "signal", sig.String())
		cancel()
	}()

	return ctx, cancel
}
