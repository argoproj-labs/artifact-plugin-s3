.PHONY: proto build clean run test lint

# Default target
all: artifact-server

# Download proto files
proto/google/protobuf/descriptor.proto:
	@echo "Downloading Google protobuf descriptor.proto..."
	@mkdir -p proto/google/protobuf
	@curl -s -o $@ https://raw.githubusercontent.com/protocolbuffers/protobuf/main/src/google/protobuf/descriptor.proto

proto/google/api/annotations.proto:
	@echo "Downloading Google API annotations.proto..."
	@mkdir -p proto/google/api
	@curl -s -o $@ https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/annotations.proto

proto/google/api/http.proto:
	@echo "Downloading Google API http.proto..."
	@mkdir -p proto/google/api
	@curl -s -o $@ https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/http.proto

proto/artifact.proto: proto/google/protobuf/descriptor.proto proto/google/api/annotations.proto proto/google/api/http.proto
	@echo "Downloading artifact.proto..."
	@mkdir -p proto
	@curl -s -o $@ https://raw.githubusercontent.com/pipekit/argo-workflows/refs/heads/artifact-plugins/pkg/apiclient/artifact/artifact.proto

GENERATED_GO := pkg/artifact/artifact.pb.go pkg/artifact/artifact_grpc.pb.go
# Generate Go code from proto
pkg/artifact/artifact.pb.go: proto/artifact.proto Makefile
	@echo "Generating Go code from proto..."
	@mkdir -p pkg/artifact
	@protoc -I proto --go_out=pkg/artifact --go_opt=paths=source_relative --go_opt=Mproto/artifact.proto=. $<

pkg/artifact/artifact_grpc.pb.go: proto/artifact.proto Makefile
	@echo "Generating gRPC Go code from proto..."
	@mkdir -p pkg/artifact
	@protoc -I proto --go-grpc_out=pkg/artifact --go-grpc_opt=paths=source_relative --go-grpc_opt=Mproto/artifact.proto=. $<

# Build the binary
artifact-server: lint test  $(GENERATED_GO) main.go
	@echo "Building S3 artifact plugin server..."
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" -o $@ main.go

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf proto/
	@rm -rf pkg/artifact/
	@rm -f artifact-server

# Run the server (requires socket path argument)
run: artifact-server
	@if [ -z "$(SOCKET)" ]; then \
		echo "Usage: make run SOCKET=/path/to/socket"; \
		exit 1; \
	fi
	@echo "Starting S3 artifact plugin server on $(SOCKET)..."
	@./artifact-server $(SOCKET)

test: $(GENERATED_GO)
	@echo "Running tests..."
	@go test ./...

lint: $(GENERATED_GO)
	@echo "Running lint..."
	@go vet ./...
	@go fmt ./...

DOCKER_TAG ?= latest
DOCKER_IMAGE ?= ghcr.io/argoproj-labs/artifact-driver-s3:$(DOCKER_TAG)

docker-build: $(GENERATED_GO)
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE) .

docker-push: docker-build
	@echo "Pushing Docker image..."
	@docker push $(DOCKER_IMAGE)
