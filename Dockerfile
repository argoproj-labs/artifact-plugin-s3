# Build stage
FROM golang:1.24.4-alpine AS builder

RUN apk add --no-cache \
    make \
    protobuf \
    curl

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Copy source code
COPY . .

# Build the binary
RUN make -j 4 artifact-server

# Runtime stage
FROM gcr.io/distroless/static

USER 8737

# Copy the binary from builder stage
COPY --chown=8737 --from=builder /app/artifact-server /artifact-server

# Set the binary as entrypoint
ENTRYPOINT ["/artifact-server"]
