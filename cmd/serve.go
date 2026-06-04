package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ParthSareen/zuko/remote"
	"github.com/spf13/cobra"
)

var serveAddr string
var serveTailscale bool

func init() {
	serveCmd.Flags().StringVar(&serveAddr, "addr", "127.0.0.1:9777", "listen address")
	serveCmd.Flags().BoolVar(&serveTailscale, "tailscale", false, "bind to this Mac's Tailscale IPv4 address")
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve remote approval requests",
	Long:  "Runs a small HTTP server for paired remote clients to approve locked zuko commands without exposing anything publicly.",
	RunE:  runServe,
}

func runServe(_ *cobra.Command, _ []string) error {
	addr, err := resolveServeAddr(serveAddr, serveTailscale)
	if err != nil {
		return err
	}

	localToken, err := remote.NewToken()
	if err != nil {
		return fmt.Errorf("failed to create local token: %w", err)
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	defer listener.Close()

	url := "http://" + listenerHostPort(listener.Addr())
	state := remote.ServeState{
		URL:        url,
		LocalToken: localToken,
		PID:        os.Getpid(),
		StartedAt:  time.Now(),
	}
	if err := remote.SaveServeState(state); err != nil {
		return fmt.Errorf("failed to save serve state: %w", err)
	}
	defer remote.RemoveServeState(localToken)

	server := &http.Server{
		Handler:           remote.NewServer(localToken),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		errCh <- err
	}()

	fmt.Printf("Zuko approval server listening on %s\n", url)
	if serveTailscale {
		fmt.Println("Bound to Tailscale only.")
	}
	fmt.Println(`Pair a client with: zuko pair "Apple Watch"`)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	case err := <-errCh:
		return err
	}
}

func resolveServeAddr(addr string, tailscale bool) (string, error) {
	if !tailscale {
		return addr, nil
	}

	_, port, err := net.SplitHostPort(addr)
	if err != nil || port == "" {
		port = "9777"
	}

	ip, err := tailscaleIPv4()
	if err != nil {
		return "", err
	}
	return net.JoinHostPort(ip, port), nil
}

func tailscaleIPv4() (string, error) {
	out, err := exec.Command("tailscale", "ip", "-4").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Tailscale IPv4 with 'tailscale ip -4': %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		ip := strings.TrimSpace(line)
		if ip == "" {
			continue
		}
		if parsed := net.ParseIP(ip); parsed != nil && parsed.To4() != nil {
			return ip, nil
		}
	}
	return "", fmt.Errorf("tailscale ip -4 did not return an IPv4 address")
}

func listenerHostPort(addr net.Addr) string {
	tcpAddr, ok := addr.(*net.TCPAddr)
	if !ok {
		return addr.String()
	}
	host := tcpAddr.IP.String()
	if host == "" || host == "::" {
		host = "localhost"
	}
	return net.JoinHostPort(host, fmt.Sprintf("%d", tcpAddr.Port))
}
