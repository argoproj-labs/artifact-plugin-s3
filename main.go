package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/pipekit/artifact-plugin-s3/pkg/artifact"
	"github.com/pipekit/artifact-plugin-s3/pkg/s3"
)

type artifactServer struct {
	artifact.UnimplementedArtifactServiceServer
}

const (
	logLevel  = logging.Debug
	logFormat = logging.JSON
)

var logger = logging.NewSlogLogger(logLevel, logFormat)

// validatePluginArtifact validates that an artifact has proper plugin configuration
func validatePluginArtifact(artifact *artifact.Artifact) error {
	if artifact == nil {
		return status.Error(codes.InvalidArgument, "artifact is required")
	}

	if artifact.Plugin == nil {
		return status.Error(codes.InvalidArgument, "plugin artifact location is required")
	}

	if artifact.Plugin.Configuration == "" {
		return status.Error(codes.InvalidArgument, "plugin configuration is required")
	}

	return nil
}

// getDriver extracts and validates plugin configuration from an artifact
func getDriver(ctx context.Context, artifact *artifact.Artifact) (*s3.ArtifactDriver, *wfv1.Artifact, error) {
	if err := validatePluginArtifact(artifact); err != nil {
		return nil, nil, err
	}

	pluginArtifact := artifact.Plugin

	// Resolve S3 configuration and credentials
	driver, argoArtifact, err := s3.DriverAndArtifactFromConfig(ctx, pluginArtifact.Configuration, pluginArtifact.Key)
	if err != nil {
		return nil, nil, status.Error(codes.Internal, err.Error())
	}

	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithField("driver", driver).Info(ctx, "Created S3 driver")
	logger.WithField("artifact", argoArtifact).Info(ctx, "Created Argo artifact")
	return driver, argoArtifact, nil
}

func (s *artifactServer) Load(ctx context.Context, req *artifact.LoadArtifactRequest) (*artifact.LoadArtifactResponse, error) {
	ctx = logging.WithLogger(ctx, logger)
	logger.WithField("request", req).Info(ctx, "Load artifact request")

	if req.InputArtifact == nil {
		return &artifact.LoadArtifactResponse{
			Success: false,
			Error:   "input artifact is required",
		}, nil
	}

	driver, argoArtifact, err := getDriver(ctx, req.InputArtifact)
	if err != nil {
		return &artifact.LoadArtifactResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Load the artifact
	err = driver.Load(ctx, argoArtifact, req.Path)
	if err != nil {
		return &artifact.LoadArtifactResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &artifact.LoadArtifactResponse{
		Success: true,
	}, nil
}

func (s *artifactServer) OpenStream(req *artifact.OpenStreamRequest, stream artifact.ArtifactService_OpenStreamServer) error {
	ctx := logging.WithLogger(stream.Context(), logger)
	logger.WithField("request", req).Info(ctx, "Open stream request")

	driver, argoArtifact, err := getDriver(ctx, req.Artifact)
	if err != nil {
		return err
	}

	// Open stream
	reader, err := driver.OpenStream(ctx, argoArtifact)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	defer reader.Close()

	// Stream data in chunks
	buffer := make([]byte, 1024*1024) // 1MB chunks
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			response := &artifact.OpenStreamResponse{
				Data:  buffer[:n],
				IsEnd: false,
			}
			if err := stream.Send(response); err != nil {
				return status.Error(codes.Internal, err.Error())
			}
		}
		if err != nil {
			break
		}
	}

	// Send end marker
	response := &artifact.OpenStreamResponse{
		Data:  []byte{},
		IsEnd: true,
	}
	return stream.Send(response)
}

func (s *artifactServer) Save(ctx context.Context, req *artifact.SaveArtifactRequest) (*artifact.SaveArtifactResponse, error) {
	ctx = logging.WithLogger(ctx, logger)
	logger.WithField("request", req).Info(ctx, "Save artifact request")

	if req.OutputArtifact == nil {
		return &artifact.SaveArtifactResponse{
			Success: false,
			Error:   "output artifact is required",
		}, nil
	}

	driver, argoArtifact, err := getDriver(ctx, req.OutputArtifact)
	if err != nil {
		return &artifact.SaveArtifactResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Save the artifact
	err = driver.Save(ctx, req.Path, argoArtifact)
	if err != nil {
		return &artifact.SaveArtifactResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &artifact.SaveArtifactResponse{
		Success: true,
	}, nil
}

func (s *artifactServer) Delete(ctx context.Context, req *artifact.DeleteArtifactRequest) (*artifact.DeleteArtifactResponse, error) {
	ctx = logging.WithLogger(ctx, logger)
	logger.WithField("request", req).Info(ctx, "Delete artifact request")

	driver, argoArtifact, err := getDriver(ctx, req.Artifact)
	if err != nil {
		return &artifact.DeleteArtifactResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Delete the artifact
	err = driver.Delete(ctx, argoArtifact)
	if err != nil {
		return &artifact.DeleteArtifactResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &artifact.DeleteArtifactResponse{
		Success: true,
	}, nil
}

func (s *artifactServer) ListObjects(ctx context.Context, req *artifact.ListObjectsRequest) (*artifact.ListObjectsResponse, error) {
	ctx = logging.WithLogger(ctx, logger)
	logger.WithField("request", req).Info(ctx, "List objects request")

	driver, argoArtifact, err := getDriver(ctx, req.Artifact)
	if err != nil {
		return &artifact.ListObjectsResponse{
			Error: err.Error(),
		}, nil
	}

	// List objects
	objects, err := driver.ListObjects(ctx, argoArtifact)
	if err != nil {
		return &artifact.ListObjectsResponse{
			Error: err.Error(),
		}, nil
	}

	return &artifact.ListObjectsResponse{
		Objects: objects,
	}, nil
}

func (s *artifactServer) IsDirectory(ctx context.Context, req *artifact.IsDirectoryRequest) (*artifact.IsDirectoryResponse, error) {
	ctx = logging.WithLogger(ctx, logger)
	logger.WithField("request", req).Info(ctx, "Is directory request")

	driver, argoArtifact, err := getDriver(ctx, req.Artifact)
	if err != nil {
		return &artifact.IsDirectoryResponse{
			Error: err.Error(),
		}, nil
	}

	// Check if it's a directory
	isDir, err := driver.IsDirectory(ctx, argoArtifact)
	if err != nil {
		return &artifact.IsDirectoryResponse{
			Error: err.Error(),
		}, nil
	}

	return &artifact.IsDirectoryResponse{
		IsDirectory: isDir,
	}, nil
}

// startServer creates and configures the gRPC server with the artifact service,
// sets up the Unix socket listener, and returns both for the caller to manage.
// This function handles socket cleanup and directory creation but does not start
// serving - that's left to the caller.
func startServer(ctx context.Context, socketPath string) (*grpc.Server, net.Listener, error) {
	// Remove any existing socket file
	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		return nil, nil, err
	}

	// Create the Unix socket listener
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, nil, err
	}

	// Create and configure the gRPC server
	server := grpc.NewServer()
	artifact.RegisterArtifactServiceServer(server, &artifactServer{})

	return server, listener, nil
}

// parseArgs validates command line arguments and returns the socket path
func parseArgs(ctx context.Context) string {
	if len(os.Args) != 2 {
		logger.WithField("usage", "artifact-server <unix-socket-path>").WithFatal().Error(ctx, "Usage")
	}
	return os.Args[1]
}

// verifySocket checks the socket file was created properly with correct permissions
func verifySocket(ctx context.Context, socketPath string) {
	socketInfo, err := os.Stat(socketPath)
	if err != nil {
		logger.WithError(err).WithFatal().Error(ctx, "Failed to get socket file info")
	}
	logger.WithFields(logging.Fields{
		"socketPath": socketPath,
		"mode":       socketInfo.Mode().String(),
		"size":       socketInfo.Size(),
	}).Info(ctx, "Unix socket created successfully")
}

// setupSignalHandling configures graceful shutdown on SIGTERM
func setupSignalHandling(ctx context.Context, server *grpc.Server) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info(ctx, "Received SIGTERM, shutting down gracefully")
		server.GracefulStop()
	}()
}

func main() {
	ctx := logging.WithLogger(context.Background(), logger)
	socketPath := parseArgs(ctx)

	server, listener, err := startServer(ctx, socketPath)
	if err != nil {
		logger.WithError(err).WithFatal().Error(ctx, "Failed to start server")
	}
	defer listener.Close()

	verifySocket(ctx, socketPath)
	logger.WithField("socketPath", socketPath).Info(ctx, "Starting artifact plugin server")

	setupSignalHandling(ctx, server)

	// Log when server is ready to accept connections
	logger.WithField("address", listener.Addr().String()).Info(ctx, "Server ready to accept connections")

	if err := server.Serve(listener); err != nil {
		logger.WithError(err).WithFatal().Error(ctx, "Failed to serve")
	}
}
