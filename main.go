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
)

// These types will be generated from the proto file
type Artifact struct {
	Name    string            `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Path    string            `protobuf:"bytes,2,opt,name=path,proto3" json:"path,omitempty"`
	Url     string            `protobuf:"bytes,3,opt,name=url,proto3" json:"url,omitempty"`
	Options map[string]string `protobuf:"bytes,4,rep,name=options,proto3" json:"options,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

type LoadArtifactRequest struct {
	InputArtifact *Artifact `protobuf:"bytes,1,opt,name=input_artifact,json=inputArtifact,proto3" json:"input_artifact,omitempty"`
	Path          string    `protobuf:"bytes,2,opt,name=path,proto3" json:"path,omitempty"`
}

type LoadArtifactResponse struct {
	Success bool   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	Error   string `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
}

type OpenStreamRequest struct {
	Artifact *Artifact `protobuf:"bytes,1,opt,name=artifact,proto3" json:"artifact,omitempty"`
}

type OpenStreamResponse struct {
	Data  []byte `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	IsEnd bool   `protobuf:"varint,2,opt,name=is_end,json=isEnd,proto3" json:"is_end,omitempty"`
	Error string `protobuf:"bytes,3,opt,name=error,proto3" json:"error,omitempty"`
}

type SaveArtifactRequest struct {
	Path           string    `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	OutputArtifact *Artifact `protobuf:"bytes,2,opt,name=output_artifact,json=outputArtifact,proto3" json:"output_artifact,omitempty"`
}

type SaveArtifactResponse struct {
	Success bool   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	Error   string `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
}

type DeleteArtifactRequest struct {
	Artifact *Artifact `protobuf:"bytes,1,opt,name=artifact,proto3" json:"artifact,omitempty"`
}

type DeleteArtifactResponse struct {
	Success bool   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	Error   string `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
}

type ListObjectsRequest struct {
	Artifact *Artifact `protobuf:"bytes,1,opt,name=artifact,proto3" json:"artifact,omitempty"`
}

type ListObjectsResponse struct {
	Objects []string `protobuf:"bytes,1,rep,name=objects,proto3" json:"objects,omitempty"`
	Error   string   `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
}

type IsDirectoryRequest struct {
	Artifact *Artifact `protobuf:"bytes,1,opt,name=artifact,proto3" json:"artifact,omitempty"`
}

type IsDirectoryResponse struct {
	IsDirectory bool   `protobuf:"varint,1,opt,name=is_directory,json=isDirectory,proto3" json:"is_directory,omitempty"`
	Error       string `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
}

// Service interface
type ArtifactServiceServer interface {
	Load(context.Context, *LoadArtifactRequest) (*LoadArtifactResponse, error)
	OpenStream(*OpenStreamRequest, ArtifactService_OpenStreamServer) error
	Save(context.Context, *SaveArtifactRequest) (*SaveArtifactResponse, error)
	Delete(context.Context, *DeleteArtifactRequest) (*DeleteArtifactResponse, error)
	ListObjects(context.Context, *ListObjectsRequest) (*ListObjectsResponse, error)
	IsDirectory(context.Context, *IsDirectoryRequest) (*IsDirectoryResponse, error)
}

type ArtifactService_OpenStreamServer interface {
	Send(*OpenStreamResponse) error
	grpc.ServerStream
}

type UnimplementedArtifactServiceServer struct{}

func (UnimplementedArtifactServiceServer) Load(context.Context, *LoadArtifactRequest) (*LoadArtifactResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Load not implemented")
}

func (UnimplementedArtifactServiceServer) OpenStream(*OpenStreamRequest, ArtifactService_OpenStreamServer) error {
	return status.Errorf(codes.Unimplemented, "method OpenStream not implemented")
}

func (UnimplementedArtifactServiceServer) Save(context.Context, *SaveArtifactRequest) (*SaveArtifactResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Save not implemented")
}

func (UnimplementedArtifactServiceServer) Delete(context.Context, *DeleteArtifactRequest) (*DeleteArtifactResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Delete not implemented")
}

func (UnimplementedArtifactServiceServer) ListObjects(context.Context, *ListObjectsRequest) (*ListObjectsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListObjects not implemented")
}

func (UnimplementedArtifactServiceServer) IsDirectory(context.Context, *IsDirectoryRequest) (*IsDirectoryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method IsDirectory not implemented")
}

type artifactServer struct {
	UnimplementedArtifactServiceServer
}

func (s *artifactServer) Load(ctx context.Context, req *LoadArtifactRequest) (*LoadArtifactResponse, error) {
	log.Printf("Load artifact request: %+v", req)

	if req.InputArtifact == nil {
		return &LoadArtifactResponse{
			Success: false,
			Error:   "input artifact is required",
		}, nil
	}

	log.Printf("Loading artifact from %s to %s", req.InputArtifact.Url, req.Path)

	return &LoadArtifactResponse{
		Success: true,
	}, nil
}

func (s *artifactServer) OpenStream(req *OpenStreamRequest, stream ArtifactService_OpenStreamServer) error {
	log.Printf("Open stream request: %+v", req)

	if req.Artifact == nil {
		return status.Error(codes.InvalidArgument, "artifact is required")
	}

	log.Printf("Opening stream for artifact: %s", req.Artifact.Name)

	response := &OpenStreamResponse{
		Data:  []byte("dummy data"),
		IsEnd: true,
	}

	return stream.Send(response)
}

func (s *artifactServer) Save(ctx context.Context, req *SaveArtifactRequest) (*SaveArtifactResponse, error) {
	log.Printf("Save artifact request: %+v", req)

	if req.OutputArtifact == nil {
		return &SaveArtifactResponse{
			Success: false,
			Error:   "output artifact is required",
		}, nil
	}

	log.Printf("Saving artifact from %s to %s", req.Path, req.OutputArtifact.Url)

	return &SaveArtifactResponse{
		Success: true,
	}, nil
}

func (s *artifactServer) Delete(ctx context.Context, req *DeleteArtifactRequest) (*DeleteArtifactResponse, error) {
	log.Printf("Delete artifact request: %+v", req)

	if req.Artifact == nil {
		return &DeleteArtifactResponse{
			Success: false,
			Error:   "artifact is required",
		}, nil
	}

	log.Printf("Deleting artifact: %s", req.Artifact.Name)

	return &DeleteArtifactResponse{
		Success: true,
	}, nil
}

func (s *artifactServer) ListObjects(ctx context.Context, req *ListObjectsRequest) (*ListObjectsResponse, error) {
	log.Printf("List objects request: %+v", req)

	if req.Artifact == nil {
		return &ListObjectsResponse{
			Error: "artifact is required",
		}, nil
	}

	log.Printf("Listing objects for artifact: %s", req.Artifact.Name)

	return &ListObjectsResponse{
		Objects: []string{"object1", "object2", "object3"},
	}, nil
}

func (s *artifactServer) IsDirectory(ctx context.Context, req *IsDirectoryRequest) (*IsDirectoryResponse, error) {
	log.Printf("Is directory request: %+v", req)

	if req.Artifact == nil {
		return &IsDirectoryResponse{
			Error: "artifact is required",
		}, nil
	}

	log.Printf("Checking if artifact is directory: %s", req.Artifact.Name)

	return &IsDirectoryResponse{
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
	RegisterArtifactServiceServer(server, &artifactServer{})

	log.Printf("Starting artifact plugin server on %s", socketPath)

	if err := server.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// Placeholder function that will be replaced by generated code
func RegisterArtifactServiceServer(s *grpc.Server, srv ArtifactServiceServer) {
	// This will be implemented by the generated protobuf code
}
