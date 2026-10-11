{
  description = "Phasionary - Terminal-first project planning tool";

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
        in
        {
          phasionary = pkgs.buildGoModule {
            pname = "phasionary";
            inherit version;
            src = ./.;
            vendorHash = "sha256-VIeI6W8ybid5Bq6nIp0K4DPfZVLomE486Je8dMPmwjc=";
            subPackages = [ "cmd/phasionary" ];
            ldflags = [
              "-s"
              "-w"
              "-X phasionary/internal/version.Version=v${version}"
              "-X phasionary/internal/version.Commit=${commit}"
              "-X phasionary/internal/version.BuildDate=${buildDate}"
            ];
            nativeBuildInputs = [ pkgs.installShellFiles ];
            postInstall = ''
              installShellCompletion --cmd phasionary \
                --bash <($out/bin/phasionary completion bash) \
                --zsh <($out/bin/phasionary completion zsh) \
                --fish <($out/bin/phasionary completion fish)
            '';
            meta = {
              description = "Terminal-first, single-user project planning tool";
              homepage = "https://github.com/ilyasturki/phasionary";
              license = pkgs.lib.licenses.mit;
              mainProgram = "phasionary";
              platforms = pkgs.lib.platforms.linux ++ pkgs.lib.platforms.darwin;
            };
          };
          default = self.packages.${system}.phasionary;
        }
      );

      apps = forAllSystems (system: {
        default = {
          type = "app";
          program = "${self.packages.${system}.phasionary}/bin/phasionary";
          meta.description = "Terminal-first, single-user project planning tool";
        };
      });
    };
}
