{
  description = "Client for the hush-hush secrets object store";

  inputs = {
    # go.mod pins go 1.27.0 - only available as the pkgs.go_1_27 attribute
    # here, not the default pkgs.go (1.26.7 on this channel at the time
    # this was written). Pin nixpkgs to whatever channel actually carries
    # the toolchain version this repo needs, not reflexively the latest
    # stable release (rules/packaging.md).
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = (pkgs.buildGoModule.override { go = pkgs.go_1_27; }) {
          pname = "hush-hush-cli";
          version = self.shortRev or self.dirtyShortRev or "dev";
          src = ./.;

          # Resolved by actually building, not guessed - there's no way to
          # compute it without one (rules/packaging.md).
          vendorHash = "sha256-DEGpvEfwl86ZSAH0tN0IMNvsaLaJaLnr+OtXHc+lsdY=";

          subPackages = [ "cmd/hush-hush-cli" ];

          ldflags = [
            "-s"
            "-w"
            "-X main.version=${self.shortRev or self.dirtyShortRev or "dev"}"
          ];

          nativeBuildInputs = [ pkgs.installShellFiles ];

          # Same shape goreleaser's own before.hooks uses: generate man
          # pages by running the freshly built binary against its own
          # hidden `man <dir>` command, then installShellFiles picks up
          # the result. Only safe because a flake builds natively per
          # system (no cross-compilation) - running the just-built binary
          # during the build is fine for the same reason it's fine in a
          # goreleaser before-hook (rules/packaging.md).
          postInstall = ''
            $out/bin/hush-hush-cli man manpages
            installManPage manpages/hush-hush-cli.1
            installManPage manpages/hush-hush-cli-init.1
            installManPage manpages/hush-hush-cli-inject.1
            installManPage manpages/hush-hush-cli-get.1
            installManPage manpages/hush-hush-cli-update.1
            installManPage manpages/hush-hush-cli-delete.1
          '';

          meta = {
            description = "Client for the hush-hush secrets object store";
            homepage = "https://github.com/alrayyes/hush-hush-cli";
            license = pkgs.lib.licenses.gpl3Only;
            mainProgram = "hush-hush-cli";
          };
        };

        apps.default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/hush-hush-cli";
          meta.description = "Client for the hush-hush secrets object store";
        };

        devShells.default = pkgs.mkShell {
          packages = [ pkgs.go_1_27 ];
        };
      }
    );
}
