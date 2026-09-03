# Installation

Pick whichever fits:

- **Arch Linux (AUR)**: `paru -S hush-hush-cli-bin` (or any other AUR
  helper) - a prebuilt binary, not built from source.
- **Debian/Ubuntu**: download the `.deb` from the
  [latest release](https://github.com/alrayyes/hush-hush-cli/releases/latest)
  and `sudo dpkg -i hush-hush-cli_*.deb`.
- **Fedora/RHEL**: download the `.rpm` from the same release page and
  `sudo rpm -i hush-hush-cli_*.rpm`.
- **Any Go toolchain**:

  ```sh
  go install github.com/alrayyes/hush-hush-cli/cmd/hush-hush-cli@latest
  ```

- **Anywhere else, or from source**:

  ```sh
  git clone https://github.com/alrayyes/hush-hush-cli
  cd hush-hush-cli
  go build -o hush-hush-cli ./cmd/hush-hush-cli
  ```

Every path except `go install` installs a man page too - `man
hush-hush-cli` once it's on `PATH`. None of these are a hosted apt/dnf
repository - each is a downloadable file attached to the GitHub release,
not something `apt update` discovers on its own.

No Docker image or Nix flake yet - see
[hush-hush-cli#7](https://github.com/alrayyes/hush-hush-cli/issues/7) and
[#8](https://github.com/alrayyes/hush-hush-cli/issues/8).
