# Sample Argo Workflows Artifact Plugin Server

This plugin is not maintained or supported.
It is here as a sample implementation and used to test the Argo Workflows artifact plugin feature.

This is a gRPC server that implements the Argo Workflows artifact service interface.
It provides a basic implementation of all required artifact operations, copied from the builtin S3 artifact driver in the Argo Workflows repository.

The docker image is published as [ghcr.io/argoproj-labs/artifact-driver-s3](https://github.com/argoproj-labs/artifact-plugin-s3/pkgs/container/artifact-driver-s3).

## Prerequisites

- Go 1.24.4 or later
- Protocol Buffers compiler (`protoc`)
- Make
- Docker

## Installation

## Development Environment

This project uses [devenv](https://devenv.sh/) to manage the development environment.

### Setup

1. Install devenv:

```bash
nix profile install --accept-flake-config github:cachix/devenv/latest
```

2. Install the project dependencies:


```bash
devenv up
```

### Build

1. Build the project:
```bash
make artifact-server
```

This will:
- Download the artifact.proto file from the Argo Workflows repository
- Generate Go code from the protobuf definitions
- Build the artifact-server binary

## Usage

Run the server with a Unix socket path:

```bash
make run SOCKET=/tmp/artifact-server.sock
```

Or run the binary directly:

```bash
./artifact-server /tmp/artifact-server.sock
```

## Implementation

The server implements all methods defined in the Argo Workflows artifact service:

- `Load`: Load artifacts from a remote location
- `OpenStream`: Stream artifact data
- `Save`: Save artifacts to a remote location
- `Delete`: Delete artifacts
- `ListObjects`: List objects in an artifact location
- `IsDirectory`: Check if an artifact is a directory

## Docker

Build the Docker image:

```bash
docker build -t artifact-server .
```

Run the container:

```bash
docker run --rm -v /tmp:/tmp artifact-server /tmp/artifact-server.sock
```

The container uses a minimal `scratch` base image for security.
The binary is statically linked and requires no external libraries.

## Development

To clean build artifacts:

```bash
make clean
```
