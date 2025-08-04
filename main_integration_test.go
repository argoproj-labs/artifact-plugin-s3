package main

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/pipekit/artifact-plugin-s3/pkg/artifact"
)

// TestArtifactPluginServer_EndToEnd spins up the real artifact plugin server
// (using startServer) and proves that a client can establish a
// connection over a Unix domain socket and that the connection transitions to
// the Idle state.
func TestArtifactPluginServer_EndToEnd(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "artifact-plugin.sock")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use the actual startServer function from main.go
	srv, lis, err := startServer(ctx, socketPath)
	if err != nil {
		t.Fatalf("failed to start artifact plugin server: %v", err)
	}

	// Start serving in the background
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- srv.Serve(lis)
	}()

	// Ensure cleanup
	t.Cleanup(func() {
		srv.Stop()
		_ = lis.Close()
		// Verify the serve goroutine completed
		select {
		case serveErr := <-serveDone:
			if serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
				t.Errorf("server exited with unexpected error: %v", serveErr)
			}
		case <-time.After(1 * time.Second):
			t.Error("server did not stop within timeout")
		}
	})

	// Create a client connection using the new API
	conn, err := grpc.NewClient(
		socketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return net.DialTimeout("unix", addr, 2*time.Second)
		}),
	)
	if err != nil {
		t.Fatalf("failed to create grpc client: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	// Create the artifact service client
	client := artifact.NewArtifactServiceClient(conn)
	if client == nil {
		t.Fatal("failed to create artifact service client")
	}

	// Explicitly trigger connection
	conn.Connect()

	// Wait for the connection to be ready within the timeout.
	if !conn.WaitForStateChange(ctx, connectivity.Idle) {
		_ = conn.Close()
		t.Fatalf("connection did not reach Idle state within timeout, final state: %v", conn.GetState())
	}
	t.Logf("connection reached Idle state")
}
