{
  description = "Astra-Service — reproducible development shell and CI build inputs";

  inputs = {
    # nixos-24.11 supplies Node 22, Go 1.22, PostgreSQL 16, Redis 7 and NATS.
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.11";
    flake-utils.url = "github:numtide/flake-utils";
    rust-overlay = {
      url = "github:oxalica/rust-overlay";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, flake-utils, rust-overlay }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        overlays = [ (import rust-overlay) ];
        pkgs = import nixpkgs {
          inherit system overlays;
          config.allowUnfree = true;
        };

        # Rust 1.79.0 with the tools needed by the workspace.
        rustToolchain = pkgs.rust-bin.stable."1.79.0".default.override {
          extensions = [ "rust-src" "rustfmt" "clippy" ];
        };
      in
      {
        # `nix develop` enters the default shell.
        devShells.default = pkgs.mkShell {
          name = "astra-service-dev";

          buildInputs = with pkgs; [
            # ── Node / TypeScript toolchain ──
            nodejs_22
            pnpm
            yarn
            nodePackages.typescript
            nodePackages.ts-node

            # ── Go toolchain ──
            go_1_22
            go-tools
            golangci-lint
            govulncheck

            # ── Rust toolchain ──
            rustToolchain
            cargo-audit
            cargo-deny
            cargo-nextest

            # ── Databases / message bus ──
            postgresql_16
            redis
            nats-server
            natscli

            # ── Containers / orchestration ──
            docker
            docker-compose
            kubectl
            skaffold

            # ── Protocol / codegen ──
            protobuf
            protoc-gen-go
            protoc-gen-go-grpc
            buf

            # ── Security / supply chain ──
            syft
            cosign

            # ── Observability helpers ──
            jq
            curl
            websocat

            # ── Misc ──
            git
            gnumake
            gnused
            gnugrep
            which
            coreutils
            findutils
            nixfmt-rfc-style
          ];

          shellHook = ''
            export ASTRA_ROOT="$(pwd)"
            export PATH="$ASTRA_ROOT/.toolchain/go/bin:$ASTRA_ROOT/.toolchain/protoc/bin:$PATH"
            export GOPATH="$ASTRA_ROOT/.go"
            export GOROOT="${pkgs.go_1_22}/share/go"
            export CARGO_HOME="$ASTRA_ROOT/.cargo"
            export RUSTUP_HOME="$ASTRA_ROOT/.rustup"
            export PGDATA="$ASTRA_ROOT/.postgres"
            export REDIS_DATA="$ASTRA_ROOT/.redis"
            export NATS_DATA="$ASTRA_ROOT/.nats"

            echo "Astra-Service dev shell"
            echo "  node     $(node --version)"
            echo "  pnpm     $(pnpm --version)"
            echo "  go       $(go version)"
            echo "  rustc    $(rustc --version)"
            echo "  cargo    $(cargo --version)"
            echo "  psql     $(psql --version | head -n1)"
            echo "  redis    $(redis-server --version | head -n1)"
            echo "  nats     $(nats-server --version)"
          '';
        };

        # Convenience package set exposed for CI matrix caching.
        packages = {
          inherit rustToolchain;
          node = pkgs.nodejs_22;
          go = pkgs.go_1_22;
          postgres = pkgs.postgresql_16;
          redis = pkgs.redis;
          nats = pkgs.nats-server;
        };

        # Lightweight flake sanity checks.
        checks = {
          # Verify the requested toolchain versions resolve and run.
          toolchainVersions = pkgs.runCommand "toolchain-versions" {
            nativeBuildInputs = [
              pkgs.nodejs_22
              pkgs.go_1_22
              rustToolchain
              pkgs.postgresql_16
            ];
          } ''
            echo "Node:  $(node --version)" > $out
            echo "Go:    $(go version)" >> $out
            echo "Rust:  $(rustc --version)" >> $out
            echo "PG:    $(psql --version)" >> $out
          '';
        };
      });
}
