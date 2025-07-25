package main

import (
	"context"
	"log"
	"net"
	"os"
	"path/filepath"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pipekit/artifact-plugin-s3/pkg/artifact"
	"github.com/pipekit/artifact-plugin-s3/pkg/s3"
)

type artifactServer struct {
	artifact.UnimplementedArtifactServiceServer
}

func (s *artifactServer) Load(ctx context.Context, req *artifact.LoadArtifactRequest) (*artifact.LoadArtifactResponse, error) {
	log.Printf("Load artifact request: %+v", req)

	if req.InputArtifact == nil {
		return &artifact.LoadArtifactResponse{
			Success: false,
			Error:   "input artifact is required",
		}, nil
	}

	if req.InputArtifact.ArtifactLocation == nil || req.InputArtifact.ArtifactLocation.S3 == nil {
		return &artifact.LoadArtifactResponse{
			Success: false,
			Error:   "S3 artifact location is required",
		}, nil
	}

	// Resolve S3 configuration and credentials
	s3Config, err := s3.ResolveCredentials(ctx, req.InputArtifact.ArtifactLocation.S3)
	if err != nil {
		return &artifact.LoadArtifactResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Create S3 driver
	driver := s3.CreateArtifactDriver(s3Config)

	// Convert to Argo artifact format
	argoArtifact := s3.ConvertToArgoArtifact(req.InputArtifact)

	// Load the artifact
	err = driver.Load(argoArtifact, req.Path)
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
	log.Printf("Open stream request: %+v", req)

	if req.Artifact == nil {
		return status.Error(codes.InvalidArgument, "artifact is required")
	}

	if req.Artifact.ArtifactLocation == nil || req.Artifact.ArtifactLocation.S3 == nil {
		return status.Error(codes.InvalidArgument, "S3 artifact location is required")
	}

	// Resolve S3 configuration and credentials
	s3Config, err := s3.ResolveCredentials(stream.Context(), req.Artifact.ArtifactLocation.S3)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	// Create S3 driver
	driver := s3.CreateArtifactDriver(s3Config)

	// Convert to Argo artifact format
	argoArtifact := s3.ConvertToArgoArtifact(req.Artifact)

	// Open stream
	reader, err := driver.OpenStream(argoArtifact)
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
	log.Printf("Save artifact request: %+v", req)

	if req.OutputArtifact == nil {
		return &artifact.SaveArtifactResponse{
			Success: false,
			Error:   "output artifact is required",
		}, nil
	}

	if req.OutputArtifact.ArtifactLocation == nil || req.OutputArtifact.ArtifactLocation.S3 == nil {
		return &artifact.SaveArtifactResponse{
			Success: false,
			Error:   "S3 artifact location is required",
		}, nil
	}

	// Resolve S3 configuration and credentials
	s3Config, err := s3.ResolveCredentials(ctx, req.OutputArtifact.ArtifactLocation.S3)
	if err != nil {
		return &artifact.SaveArtifactResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Create S3 driver
	driver := s3.CreateArtifactDriver(s3Config)

	// Convert to Argo artifact format
	argoArtifact := s3.ConvertToArgoArtifact(req.OutputArtifact)

	// Save the artifact
	err = driver.Save(req.Path, argoArtifact)
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
	log.Printf("Delete artifact request: %+v", req)

	if req.Artifact == nil {
		return &artifact.DeleteArtifactResponse{
			Success: false,
			Error:   "artifact is required",
		}, nil
	}

	if req.Artifact.ArtifactLocation == nil || req.Artifact.ArtifactLocation.S3 == nil {
		return &artifact.DeleteArtifactResponse{
			Success: false,
			Error:   "S3 artifact location is required",
		}, nil
	}

	// Resolve S3 configuration and credentials
	s3Config, err := s3.ResolveCredentials(ctx, req.Artifact.ArtifactLocation.S3)
	if err != nil {
		return &artifact.DeleteArtifactResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Create S3 driver
	driver := s3.CreateArtifactDriver(s3Config)

	// Convert to Argo artifact format
	argoArtifact := s3.ConvertToArgoArtifact(req.Artifact)

	// Delete the artifact
	err = driver.Delete(argoArtifact)
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
	log.Printf("List objects request: %+v", req)

	if req.Artifact == nil {
		return &artifact.ListObjectsResponse{
			Error: "artifact is required",
		}, nil
	}

	if req.Artifact.ArtifactLocation == nil || req.Artifact.ArtifactLocation.S3 == nil {
		return &artifact.ListObjectsResponse{
			Error: "S3 artifact location is required",
		}, nil
	}

	// Resolve S3 configuration and credentials
	s3Config, err := s3.ResolveCredentials(ctx, req.Artifact.ArtifactLocation.S3)
	if err != nil {
		return &artifact.ListObjectsResponse{
			Error: err.Error(),
		}, nil
	}

	// Create S3 driver
	driver := s3.CreateArtifactDriver(s3Config)

	// Convert to Argo artifact format
	argoArtifact := s3.ConvertToArgoArtifact(req.Artifact)

	// List objects
	objects, err := driver.ListObjects(argoArtifact)
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
	log.Printf("Is directory request: %+v", req)

	if req.Artifact == nil {
		return &artifact.IsDirectoryResponse{
			Error: "artifact is required",
		}, nil
	}

	if req.Artifact.ArtifactLocation == nil || req.Artifact.ArtifactLocation.S3 == nil {
		return &artifact.IsDirectoryResponse{
			Error: "S3 artifact location is required",
		}, nil
	}

	// Resolve S3 configuration and credentials
	s3Config, err := s3.ResolveCredentials(ctx, req.Artifact.ArtifactLocation.S3)
	if err != nil {
		return &artifact.IsDirectoryResponse{
			Error: err.Error(),
		}, nil
	}

	// Create S3 driver
	driver := s3.CreateArtifactDriver(s3Config)

	// Convert to Argo artifact format
	argoArtifact := s3.ConvertToArgoArtifact(req.Artifact)

	// Check if it's a directory
	isDir, err := driver.IsDirectory(argoArtifact)
	if err != nil {
		return &artifact.IsDirectoryResponse{
			Error: err.Error(),
		}, nil
	}

	return &artifact.IsDirectoryResponse{
		IsDirectory: isDir,
	}, nil
}

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: artifact-server <unix-socket-path>")
	}

	socketPath := os.Args[1]

	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		log.Fatalf("Failed to remove existing socket file: %v", err)
	}

	socketDir := filepath.Dir(socketPath)
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		log.Fatalf("Failed to create socket directory: %v", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatalf("Failed to listen on socket %s: %v", socketPath, err)
	}
	defer listener.Close()

	server := grpc.NewServer()
	artifact.RegisterArtifactServiceServer(server, &artifactServer{})

	log.Printf("Starting artifact plugin server on %s", socketPath)

	if err := server.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
