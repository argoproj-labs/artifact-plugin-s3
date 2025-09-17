package main

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/pipekit/artifact-plugin-s3/pkg/artifact"
)

// TestServerStartAndConnectUnixSocket spins up the gRPC server on a Unix domain socket and
// verifies that a client can connect and the connection eventually transitions to the Idle state.
func TestServerStartAndConnectUnixSocket(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for the socket file so concurrent test executions won't clash.
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "artifact-plugin.sock")

	// Start listening on the Unix socket.
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to listen on unix socket: %v", err)
	}

	// Create the gRPC server and register our service implementation.
	grpcServer := grpc.NewServer()
	artifact.RegisterArtifactServiceServer(grpcServer, &artifactServer{})

	// Start serving in the background.
	go func() {
		if serveErr := grpcServer.Serve(listener); serveErr != nil {
			// Serving should stop only when grpcServer.Stop() is called in cleanup.
			// If it exits earlier it's a test failure.
			t.Errorf("grpc server stopped unexpectedly: %v", serveErr)
		}
	}()

	// Ensure the server is stopped and resources are cleaned up when the test finishes.
	t.Cleanup(func() {
		grpcServer.Stop()
		_ = listener.Close()
		_ = os.Remove(socketPath)
	})

	// Create a client connection to the server using the new lazy-connect API.
	conn, err := grpc.NewClient(
		socketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return net.DialTimeout("unix", addr, 2*time.Second)
		}),
	)
	if err != nil {
		t.Fatalf("failed to dial unix socket: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	// Instantiate the client to prove we can create one.
	_ = artifact.NewArtifactServiceClient(conn)

	// Explicitly initiate the connection handshake.
	conn.Connect()

	// Wait for the connection to become Idle. We loop because the state may transition
	// through Connecting/Ready before it becomes Idle.
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer waitCancel()

	for {
		state := conn.GetState()
		if state == connectivity.Idle {
			return // Success: the connection reached Idle state.
		}
		if !conn.WaitForStateChange(waitCtx, state) {
			t.Fatalf("connection did not reach Idle state, final state: %v", conn.GetState())
		}
	}
}
