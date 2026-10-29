{
  description = "Ordo - Terminal-first project planning tool";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { self, nixpkgs }:
    let
      forAllSystems = nixpkgs.lib.genAttrs [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];
      version = nixpkgs.lib.fileContents ./VERSION;
      commit = self.shortRev or self.dirtyShortRev or "none";
      buildDate = self.lastModifiedDate or "unknown";
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          web = pkgs.callPackage ./nix/web.nix { inherit version; };
        in
        {
          ordo = pkgs.buildGoModule {
            pname = "ordo";
            inherit version;
            src = pkgs.lib.fileset.toSource {
              root = ./.;
              fileset = pkgs.lib.fileset.difference ./. ./native;
            };
            vendorHash = "sha256-y6cR7/CISh9WTtW+g4SfqbbSlpGcYTxoJP0cycCF4bo=";
            subPackages = [ "cmd/ordo" ];
            ldflags = [
              "-s"
              "-w"
              "-X github.com/ilyasturki/ordo/internal/version.Version=v${version}"
              "-X github.com/ilyasturki/ordo/internal/version.Commit=${commit}"
              "-X github.com/ilyasturki/ordo/internal/version.BuildDate=${buildDate}"
            ];
            nativeBuildInputs = [ pkgs.installShellFiles ];
            # Not preBuild: buildGoModule runs that in the vendor derivation too, before vendor/ exists.
            postConfigure = ''
              find internal/web/spa -mindepth 1 -maxdepth 1 ! -name placeholder.html -exec rm -rf {} +
            ''
            + pkgs.lib.optionalString (web != null) ''
              cp -R --no-preserve=mode ${web}/. internal/web/spa/
              GOOS=js GOARCH=wasm CGO_ENABLED=0 go build -o internal/web/spa/ordo-core.wasm ./cmd/ordo-core
              install -m 644 "$(go env GOROOT)/lib/wasm/wasm_exec.js" internal/web/spa/wasm_exec.js
            '';
            postInstall = ''
              installShellCompletion --cmd ordo \
                --bash <($out/bin/ordo completion bash) \
                --zsh <($out/bin/ordo completion zsh) \
                --fish <($out/bin/ordo completion fish)
            '';
            meta = {
              description = "Terminal-first, single-user project planning tool";
              homepage = "https://github.com/ilyasturki/ordo";
              license = pkgs.lib.licenses.mit;
              mainProgram = "ordo";
              platforms = pkgs.lib.platforms.linux ++ pkgs.lib.platforms.darwin;
            };
          };
          default = self.packages.${system}.ordo;
        }
        // pkgs.lib.optionalAttrs (web != null) { inherit web; }
        // pkgs.lib.optionalAttrs (system == "x86_64-linux") {
          android-sdk = import ./nix/android-sdk.nix { inherit nixpkgs system; };
        }
      );

      devShells.x86_64-linux.native = nixpkgs.legacyPackages.x86_64-linux.mkShellNoCC rec {
        ANDROID_HOME = "${self.packages.x86_64-linux.android-sdk}/libexec/android-sdk";
        # Gradle stops when ANDROID_SDK_ROOT names another SDK than ANDROID_HOME.
        ANDROID_SDK_ROOT = ANDROID_HOME;
      };

      nixosModules.default = import ./nix/module.nix { inherit self; };

      apps = forAllSystems (system: {
        default = {
          type = "app";
          program = "${self.packages.${system}.ordo}/bin/ordo";
          meta.description = "Terminal-first, single-user project planning tool";
        };
      });
    };
}
