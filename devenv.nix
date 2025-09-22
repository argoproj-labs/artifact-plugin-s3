{ pkgs, lib, config, inputs, ... }:

{
  packages = with pkgs; [
    protobuf
    protoc-gen-go
    protoc-gen-go-grpc
    gnumake
    go_1_24
    gopls
  ];

  enterShell = ''
    echo "Entering artifact-plugin-s3 shell"
    unset GOPATH;
    unset GOROOT;
  '';
}
