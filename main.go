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

	log.Printf("Loading artifact from %s to %s", req.InputArtifact.Url, req.Path)

	return &artifact.LoadArtifactResponse{
		Success: true,
	}, nil
}

func (s *artifactServer) OpenStream(req *artifact.OpenStreamRequest, stream artifact.ArtifactService_OpenStreamServer) error {
	log.Printf("Open stream request: %+v", req)

	if req.Artifact == nil {
		return status.Error(codes.InvalidArgument, "artifact is required")
	}

	log.Printf("Opening stream for artifact: %s", req.Artifact.Name)

	response := &artifact.OpenStreamResponse{
		Data:  []byte("dummy data"),
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

	log.Printf("Saving artifact from %s to %s", req.Path, req.OutputArtifact.Url)

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

	log.Printf("Deleting artifact: %s", req.Artifact.Name)

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

	log.Printf("Listing objects for artifact: %s", req.Artifact.Name)

	return &artifact.ListObjectsResponse{
		Objects: []string{"object1", "object2", "object3"},
	}, nil
}

func (s *artifactServer) IsDirectory(ctx context.Context, req *artifact.IsDirectoryRequest) (*artifact.IsDirectoryResponse, error) {
	log.Printf("Is directory request: %+v", req)

	if req.Artifact == nil {
		return &artifact.IsDirectoryResponse{
			Error: "artifact is required",
		}, nil
	}

	log.Printf("Checking if artifact is directory: %s", req.Artifact.Name)

	return &artifact.IsDirectoryResponse{
		IsDirectory: false,
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
