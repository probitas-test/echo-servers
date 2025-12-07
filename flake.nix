{
  description = "Probitas test servers development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    probitas.url = "github:jsr-probitas/probitas";
    probitas.inputs.nixpkgs.follows = "nixpkgs";
    probitas.inputs.flake-utils.follows = "flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils, probitas }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        probitasPackage = probitas.packages.${system}.probitas;
      in
      {
        devShells.default = pkgs.mkShell {
          packages = [
            # Go
            pkgs.go

            # Linting and formatting
            pkgs.golangci-lint
            pkgs.gotools # goimports

            # Protocol Buffers
            pkgs.protobuf
            pkgs.protoc-gen-go
            pkgs.protoc-gen-go-grpc
            pkgs.protoc-gen-connect-go

            # gRPC client for testing
            pkgs.grpcurl

            # Task runner
            pkgs.just

            # Formatter
            pkgs.dprint

            # Probitas scenario runner
            probitasPackage
          ];

          shellHook = ''
            echo "Probitas test servers development environment"
            echo "Go $(go version | cut -d' ' -f3)"
          '';
        };
      }
    );
}
