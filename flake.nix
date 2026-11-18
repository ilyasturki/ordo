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
      vendorHash = "sha256-YcV3OAIbedLV/SivDJIxow4qJJxfv7Fgax7TzIjNEmQ=";
      buildDate = self.lastModifiedDate or "unknown";
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          ordo = pkgs.buildGoModule {
            pname = "ordo";
            inherit version;
            src = pkgs.lib.fileset.toSource {
              root = ./.;
              fileset = pkgs.lib.fileset.difference ./. (
                pkgs.lib.fileset.unions [
                  ./native
                  ./web
                ]
              );
            };
            inherit vendorHash;
            subPackages = [ "cmd/ordo" ];
            ldflags = [
              "-s"
              "-w"
              "-X github.com/ilyasturki/ordo/internal/version.Version=v${version}"
              "-X github.com/ilyasturki/ordo/internal/version.Commit=${commit}"
              "-X github.com/ilyasturki/ordo/internal/version.BuildDate=${buildDate}"
            ];
            nativeBuildInputs = [ pkgs.installShellFiles ];
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
        // pkgs.callPackage ./nix/web.nix { inherit version vendorHash; }
        // pkgs.lib.optionalAttrs (system == "x86_64-linux") {
          android-sdk = import ./nix/android-sdk.nix { inherit nixpkgs system; };
        }
      );

      devShells = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          web = pkgs.mkShellNoCC {
            packages = [
              pkgs.bun
              pkgs.nodejs
              pkgs.go
            ];
            PLAYWRIGHT_BROWSERS_PATH = pkgs.playwright-driver.browsers;
            PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS = "true";
            # scripts/web.sh checks web/package.json pins the @playwright/test these browsers are for.
            ORDO_PLAYWRIGHT = pkgs.playwright-driver.version;
          };
        }
        // pkgs.lib.optionalAttrs (system == "x86_64-linux") {
          native = pkgs.mkShellNoCC rec {
            ANDROID_HOME = "${self.packages.x86_64-linux.android-sdk}/libexec/android-sdk";
            # Gradle stops when ANDROID_SDK_ROOT names another SDK than ANDROID_HOME.
            ANDROID_SDK_ROOT = ANDROID_HOME;
          };
        }
      );

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
