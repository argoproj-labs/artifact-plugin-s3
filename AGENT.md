# Artifact Plugin S3 - Agent Guide

## Commands
- Build: `make artifact-server` (generates protobuf, runs lint/test, builds binary)
- Test: `go test ./...` or `make test`
- Test single: `go test -run TestName ./...`
- Lint: `go vet ./... && go fmt ./...` or `make lint`
- Clean: `make clean`
- Run: `make run SOCKET=/path/to/socket` or `./artifact-server /path/to/socket`

## Architecture
- gRPC artifact plugin server for Argo Workflows S3 integration
- Main components: `/pkg/artifact/` (protobuf generated), `/pkg/s3/` (S3 driver)
- Uses Unix domain sockets for communication
- Implements: Load, Save, Delete, OpenStream, ListObjects, IsDirectory

## Code Style
- Go modules with Go 1.24.4, using modern libraries and features and no legacy code
- Import order: stdlib, third-party, local (`github.com/pipekit/artifact-plugin-s3/pkg/*`)
- Error handling: return structured responses with Success/Error fields
- Logging: slog with context from the logging package, use `logging.WithLogger(ctx, logger)`, always as structured logging
- Naming: camelCase methods, validate* prefix for validation functions
- Testing: use `t.Parallel()`, `t.TempDir()`, context with timeout, proper cleanup
