{
  description = "Probitas test servers development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
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

            # Formatters
            dprint
            nixfmt
          ];

          shellHook = ''
            echo "Probitas test servers development environment"
            echo "Go $(go version | cut -d' ' -f3)"
          '';
        };
      });

      formatter = forAllSystems (pkgs: pkgs.nixfmt);
    };
}
