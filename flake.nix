{
  description = "Clave Dev";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
      flake-utils.lib.eachDefaultSystem (system:
        let
          pkgs = import nixpkgs { inherit system; };
        in
        {
          devShells.default = pkgs.mkShell {
            packages = with pkgs; [
              # Go
              go
              gopls
              golangci-lint
              gotools

              # Node
              nodejs_20
              corepack

              # Utilities
              git
              jq
              curl
            ];

            shellHook = ''
              export GOPATH="$PWD/.gopath"
              export GOMODCACHE="$GOPATH/pkg/mod"
              export GOCACHE="$GOPATH/cache/go-build"
              export PATH="$GOPATH/bin:$PATH"

              corepack enable >/dev/null 2>&1 || true

              echo "Dev shell ready: $(go version) | node=$(node -v)"
            '';
          };
        }
      );

}
