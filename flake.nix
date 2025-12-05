{
  description = "Probitas test servers development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            # Go
            go

            # Linting and formatting
            golangci-lint
            gotools # goimports

            # Protocol Buffers
            protobuf
            protoc-gen-go
            protoc-gen-go-grpc
            protoc-gen-connect-go

            # gRPC client for testing
            grpcurl

            # Task runner
            just

            # Formatter
            dprint
          ];

          shellHook = ''
            echo "Probitas test servers development environment"
            echo "Go $(go version | cut -d' ' -f3)"
          '';
        };
      }
    );
}
