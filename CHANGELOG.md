# Changelog

## [1.4.4](https://github.com/alrayyes/hush-hush-cli/compare/v1.4.3...v1.4.4) (2026-09-10)


### Bug Fixes

* **nix:** update flake.lock ([#48](https://github.com/alrayyes/hush-hush-cli/issues/48)) ([715c4a9](https://github.com/alrayyes/hush-hush-cli/commit/715c4a972976c7f92383ef9822756d16e024edbf))

## [1.4.3](https://github.com/alrayyes/hush-hush-cli/compare/v1.4.2...v1.4.3) (2026-09-10)


### Bug Fixes

* **ci:** use RELEASE_TOKEN for flake-lock-update PR creation ([#46](https://github.com/alrayyes/hush-hush-cli/issues/46)) ([c60551c](https://github.com/alrayyes/hush-hush-cli/commit/c60551c66885e9f90ce1bfb61c079b09381979db)), closes [#45](https://github.com/alrayyes/hush-hush-cli/issues/45)

## [1.4.2](https://github.com/alrayyes/hush-hush-cli/compare/v1.4.1...v1.4.2) (2026-09-09)


### Bug Fixes

* **deps:** downgrade bun.lock to lockfileVersion 1 for Dependabot ([#40](https://github.com/alrayyes/hush-hush-cli/issues/40)) ([6eb289c](https://github.com/alrayyes/hush-hush-cli/commit/6eb289cf47a6cd5cb7081e9a94c30c795fe056eb))

## [1.4.1](https://github.com/alrayyes/hush-hush-cli/compare/v1.4.0...v1.4.1) (2026-09-09)


### Bug Fixes

* **deps:** bump the go-dependencies group with 3 updates ([#36](https://github.com/alrayyes/hush-hush-cli/issues/36)) ([9638fb3](https://github.com/alrayyes/hush-hush-cli/commit/9638fb3a0d6299d0afdea16f0357b2b1c7c108c9))

## [1.4.0](https://github.com/alrayyes/hush-hush-cli/compare/v1.3.1...v1.4.0) (2026-09-03)


### Features

* **nix:** add a flake.nix, CI verification, and flake.lock automation ([#31](https://github.com/alrayyes/hush-hush-cli/issues/31)) ([3419c39](https://github.com/alrayyes/hush-hush-cli/commit/3419c3964c293c2cbd11513766a6d3ff5b3be9fe))

## [1.3.1](https://github.com/alrayyes/hush-hush-cli/compare/v1.3.0...v1.3.1) (2026-09-03)


### Bug Fixes

* **ci:** use RELEASE_TOKEN for Dependabot auto-merge ([#28](https://github.com/alrayyes/hush-hush-cli/issues/28)) ([7ee1c7e](https://github.com/alrayyes/hush-hush-cli/commit/7ee1c7eb2bed3f2287187bd052283c4b4bf2aab1))

## [1.3.0](https://github.com/alrayyes/hush-hush-cli/compare/v1.2.1...v1.3.0) (2026-09-03)


### Features

* **docker:** add a Docker image for hush-hush-cli ([#26](https://github.com/alrayyes/hush-hush-cli/issues/26)) ([4b697a3](https://github.com/alrayyes/hush-hush-cli/commit/4b697a304eafe64e69b4881b33b3a965e115bb10))

## [1.2.1](https://github.com/alrayyes/hush-hush-cli/compare/v1.2.0...v1.2.1) (2026-09-03)


### Bug Fixes

* **packaging:** bump PKGBUILD to v1.2.0, the actual latest release ([#24](https://github.com/alrayyes/hush-hush-cli/issues/24)) ([b9915ac](https://github.com/alrayyes/hush-hush-cli/commit/b9915acc654be56c8ea6ab2ad2a6ff08b379b116))

## [1.2.0](https://github.com/alrayyes/hush-hush-cli/compare/v1.1.0...v1.2.0) (2026-09-03)


### Features

* **packaging:** add an AUR package (PKGBUILD) for hush-hush-cli-bin ([#22](https://github.com/alrayyes/hush-hush-cli/issues/22)) ([e28bc85](https://github.com/alrayyes/hush-hush-cli/commit/e28bc853a123d61ce5fc0e861ac955d44dc5a333))


### Bug Fixes

* **hooks:** run every Go hook command through pinned Docker images ([#21](https://github.com/alrayyes/hush-hush-cli/issues/21)) ([3b949e2](https://github.com/alrayyes/hush-hush-cli/commit/3b949e299c118aaf108abcb80bf3deb5abac2b03))

## [1.1.0](https://github.com/alrayyes/hush-hush-cli/compare/v1.0.0...v1.1.0) (2026-09-03)


### Features

* **release:** restore man-page generation, matching hush-hush's own wiring ([#19](https://github.com/alrayyes/hush-hush-cli/issues/19)) ([005ca33](https://github.com/alrayyes/hush-hush-cli/commit/005ca33a15edc15a5ec464d538d4403ea46dcbbd))

## 1.0.0 (2026-09-03)


### Features

* migrate the CLI from hush-hush into this repo ([#12](https://github.com/alrayyes/hush-hush-cli/issues/12)) ([0e295fc](https://github.com/alrayyes/hush-hush-cli/commit/0e295fcbb296798b48b77e8fbc12a29d6aac2b15))


### Bug Fixes

* **ci:** use the direct golangci-lint container, not the action ([#11](https://github.com/alrayyes/hush-hush-cli/issues/11)) ([4f5248d](https://github.com/alrayyes/hush-hush-cli/commit/4f5248d0b7bc7f396967c0205eadd58e1040d7b7))
* **hooks:** move golangci-lint run to pre-push, add go mod tidy -diff ([#16](https://github.com/alrayyes/hush-hush-cli/issues/16)) ([25d0722](https://github.com/alrayyes/hush-hush-cli/commit/25d0722c98d9cb64ef32cf300cac8d0d784a1c8a))
