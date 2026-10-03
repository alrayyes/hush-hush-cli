# Changelog

## [1.18.1](https://github.com/alrayyes/hush-hush-cli/compare/v1.18.0...v1.18.1) (2026-10-03)


### Bug Fixes

* **deps:** bump golang ([#208](https://github.com/alrayyes/hush-hush-cli/issues/208)) ([65dcc97](https://github.com/alrayyes/hush-hush-cli/commit/65dcc97f365e705793f94f2d7ad0d786806eba03))
* **packaging:** ship every man page in the archives and packages ([#204](https://github.com/alrayyes/hush-hush-cli/issues/204)) ([a26cddf](https://github.com/alrayyes/hush-hush-cli/commit/a26cddfe1790afdf059119ca7d8689c77d52b019)), closes [#191](https://github.com/alrayyes/hush-hush-cli/issues/191)

## [1.18.0](https://github.com/alrayyes/hush-hush-cli/compare/v1.17.0...v1.18.0) (2026-10-03)


### Features

* **cli:** add list --tag filter ([e59cb3e](https://github.com/alrayyes/hush-hush-cli/commit/e59cb3e1004c75c7c552b09409726491a465675b))

## [1.17.0](https://github.com/alrayyes/hush-hush-cli/compare/v1.16.0...v1.17.0) (2026-10-03)


### Features

* **cli:** add --keep-readable-copy to inject and update ([94078e1](https://github.com/alrayyes/hush-hush-cli/commit/94078e1a6c976a71dc460b33a498daf31fbf733a))
* **cli:** add --keep-readable-copy to inject and update ([dac5e88](https://github.com/alrayyes/hush-hush-cli/commit/dac5e882103dc6a30ae5ae681d6d3faee43b2602)), closes [#177](https://github.com/alrayyes/hush-hush-cli/issues/177)


### Bug Fixes

* **deps:** bump hush-hush-go to v4.3.0 ([1a0c659](https://github.com/alrayyes/hush-hush-cli/commit/1a0c6597c2922e762d635efdf00e7ddb837af7d1))

## [1.16.0](https://github.com/alrayyes/hush-hush-cli/compare/v1.15.0...v1.16.0) (2026-10-03)


### Features

* **cli:** add update --used-by and --clear-used-by ([ec8af15](https://github.com/alrayyes/hush-hush-cli/commit/ec8af153cce1b147a83c0f28f5b006be1ff18c7a))

## [1.15.0](https://github.com/alrayyes/hush-hush-cli/compare/v1.14.1...v1.15.0) (2026-10-03)


### Features

* **cli:** show created and updated times in list ([0b0d955](https://github.com/alrayyes/hush-hush-cli/commit/0b0d955e0addbdd029b557a5f96cf1f87447fe55))
* **cli:** show created and updated times in list ([12768a8](https://github.com/alrayyes/hush-hush-cli/commit/12768a866543dd1ea979a5f52aeaf311c9ec5dcd)), closes [#164](https://github.com/alrayyes/hush-hush-cli/issues/164)
* **cli:** show token status in token list ([c4a1ad4](https://github.com/alrayyes/hush-hush-cli/commit/c4a1ad4c5d8238662b0ecfe5e61e8c3251b6f95d)), closes [#165](https://github.com/alrayyes/hush-hush-cli/issues/165)


### Bug Fixes

* **deps:** bump hush-hush-go to v4.2.5 ([3963681](https://github.com/alrayyes/hush-hush-cli/commit/396368108a62163a454b13b330f8ded4b05b08ab))

## [1.14.1](https://github.com/alrayyes/hush-hush-cli/compare/v1.14.0...v1.14.1) (2026-10-03)


### Bug Fixes

* **ci:** ignore unpatched braces advisory in bun audit ([6f5ba47](https://github.com/alrayyes/hush-hush-cli/commit/6f5ba47a7d971133bc09db1f6d18da1d563b75fa))
* **ci:** ignore unpatched braces advisory in bun audit ([a80bb56](https://github.com/alrayyes/hush-hush-cli/commit/a80bb568dd4e55cce464438d3a5602ff7ddcf9d7)), closes [#166](https://github.com/alrayyes/hush-hush-cli/issues/166)

## [1.14.0](https://github.com/alrayyes/hush-hush-cli/compare/v1.13.0...v1.14.0) (2026-10-02)


### Features

* **cli:** support object tags on inject, update and list ([ffd9850](https://github.com/alrayyes/hush-hush-cli/commit/ffd985040b263885e4d1165d3e570b8d35e9beaa))
* **cli:** support object tags on inject, update and list ([961feda](https://github.com/alrayyes/hush-hush-cli/commit/961fedac3b5e4174c3880341d2c961835bd2fd37)), closes [#159](https://github.com/alrayyes/hush-hush-cli/issues/159)


### Bug Fixes

* **deps:** bump hush-hush-go to v4.2.3 ([76c3a6f](https://github.com/alrayyes/hush-hush-cli/commit/76c3a6fcec2bb5450f980f60cf1e7b39990f9219))

## [1.13.0](https://github.com/alrayyes/hush-hush-cli/compare/v1.12.1...v1.13.0) (2026-10-02)


### Features

* **cli:** add consumer list/add/update/delete ([35c4c78](https://github.com/alrayyes/hush-hush-cli/commit/35c4c78627ce1c35f9b8e6251fe03e742be39e9f))
* **cli:** add list --used-by filter ([e936e55](https://github.com/alrayyes/hush-hush-cli/commit/e936e5523397e3b264790a542d00b11f77a5b5ae))
* **cli:** add list --used-by filter ([ec76ba6](https://github.com/alrayyes/hush-hush-cli/commit/ec76ba6af06d506522104b29f82705f79c4674b8)), closes [#156](https://github.com/alrayyes/hush-hush-cli/issues/156)

## [1.12.1](https://github.com/alrayyes/hush-hush-cli/compare/v1.12.0...v1.12.1) (2026-10-02)


### Bug Fixes

* **deps:** bump fast-uri to 3.1.8 for GHSA-hrr3-gc8f-f4qj ([1f5c07d](https://github.com/alrayyes/hush-hush-cli/commit/1f5c07d2f1345a7a2fc4b0b73e3cff40105d41f8))
* **deps:** bump fast-uri to 3.1.8 for GHSA-hrr3-gc8f-f4qj ([51a8ade](https://github.com/alrayyes/hush-hush-cli/commit/51a8ade8ba218f666fe3ef6cc1bd5b118421f0f1)), closes [#148](https://github.com/alrayyes/hush-hush-cli/issues/148)

## [1.12.0](https://github.com/alrayyes/hush-hush-cli/compare/v1.11.0...v1.12.0) (2026-09-28)


### Features

* **cli:** add consumer read token mint/list/rotate/revoke/purge ([b13726a](https://github.com/alrayyes/hush-hush-cli/commit/b13726ad030c494c908c5b4ed361df54f667b30c))
* **cli:** add consumer read token mint/list/rotate/revoke/purge ([9733cb4](https://github.com/alrayyes/hush-hush-cli/commit/9733cb47b3d6ebb7b320dde1ac09453195ff2ebf))


### Bug Fixes

* **nix:** update vendorHash after the hush-hush-go v4.2.0 bump ([a58b199](https://github.com/alrayyes/hush-hush-cli/commit/a58b199e7c6375c3fc3268d36efd4c2583fe328c))

## [1.11.0](https://github.com/alrayyes/hush-hush-cli/compare/v1.10.2...v1.11.0) (2026-09-28)


### Features

* **cli:** add consumer read token support for get ([8e428e7](https://github.com/alrayyes/hush-hush-cli/commit/8e428e7b019123a4d8506d005a092212d10f6bff))
* **cli:** add consumer read token support for get ([ad56a59](https://github.com/alrayyes/hush-hush-cli/commit/ad56a596bba3b54bd0dca301b965969f4a5da139)), closes [#133](https://github.com/alrayyes/hush-hush-cli/issues/133)


### Bug Fixes

* **nix:** update vendorHash after the hush-hush-go v4.1.2 bump ([1d29628](https://github.com/alrayyes/hush-hush-cli/commit/1d2962826ac9d37b2fc615ee3264f369ae1d35a6))

## [1.10.2](https://github.com/alrayyes/hush-hush-cli/compare/v1.10.1...v1.10.2) (2026-09-28)


### Bug Fixes

* **integration:** send a token on reads against the real server ([055354c](https://github.com/alrayyes/hush-hush-cli/commit/055354c3ad8d1997eca9cc3047776f58baf20108))
* **integration:** send a token on reads against the real server ([9a290ef](https://github.com/alrayyes/hush-hush-cli/commit/9a290ef91428f9d3c15721da8393ed3a3f743bba)), closes [#135](https://github.com/alrayyes/hush-hush-cli/issues/135)

## [1.10.1](https://github.com/alrayyes/hush-hush-cli/compare/v1.10.0...v1.10.1) (2026-09-28)


### Bug Fixes

* **nix:** update flake.lock ([#130](https://github.com/alrayyes/hush-hush-cli/issues/130)) ([f1f0d73](https://github.com/alrayyes/hush-hush-cli/commit/f1f0d7321ce7134e83f52ab746034f25f395605d))

## [1.10.0](https://github.com/alrayyes/hush-hush-cli/compare/v1.9.0...v1.10.0) (2026-09-27)


### Features

* **inject:** resolve --used-by consumers' registered public keys ([3cd87d7](https://github.com/alrayyes/hush-hush-cli/commit/3cd87d789ceb5ce8ca23047a75e35757f5dec9ef))
* **inject:** resolve --used-by consumers' registered public keys ([858db49](https://github.com/alrayyes/hush-hush-cli/commit/858db497c8d983b7563c0966e64e07da6e276fa4)), closes [#125](https://github.com/alrayyes/hush-hush-cli/issues/125)


### Bug Fixes

* **nix:** update vendorHash after the hush-hush-go v4 bump ([1669566](https://github.com/alrayyes/hush-hush-cli/commit/1669566810bf80bfe3763e9acc4a73b265194776))

## [1.9.0](https://github.com/alrayyes/hush-hush-cli/compare/v1.8.1...v1.9.0) (2026-09-25)


### Features

* **used-by:** add a used-by command wrapping GET /objects/{id}/used-by ([#120](https://github.com/alrayyes/hush-hush-cli/issues/120)) ([e5c87ee](https://github.com/alrayyes/hush-hush-cli/commit/e5c87ee6bd81b4035c93377f4f510c849d5ca8f3))

## [1.8.1](https://github.com/alrayyes/hush-hush-cli/compare/v1.8.0...v1.8.1) (2026-09-21)


### Bug Fixes

* **nix:** update flake.lock ([#116](https://github.com/alrayyes/hush-hush-cli/issues/116)) ([f6a4125](https://github.com/alrayyes/hush-hush-cli/commit/f6a41250881aebcba973dc2b97a5bc4ea6c7806f))

## [1.8.0](https://github.com/alrayyes/hush-hush-cli/compare/v1.7.1...v1.8.0) (2026-09-20)


### Features

* **status:** add the status command ([#111](https://github.com/alrayyes/hush-hush-cli/issues/111)) ([ef92f34](https://github.com/alrayyes/hush-hush-cli/commit/ef92f3460e1284b4067c6035c8da4d742c569634))

## [1.7.1](https://github.com/alrayyes/hush-hush-cli/compare/v1.7.0...v1.7.1) (2026-09-20)


### Bug Fixes

* **deps:** bump hush-hush-go to v2.1.0 ([#108](https://github.com/alrayyes/hush-hush-cli/issues/108)) ([3844417](https://github.com/alrayyes/hush-hush-cli/commit/3844417b0eb6254964bba85e5387a4067cfd96df))

## [1.7.0](https://github.com/alrayyes/hush-hush-cli/compare/v1.6.0...v1.7.0) (2026-09-19)


### Features

* **audit-log:** add the audit-log command ([#103](https://github.com/alrayyes/hush-hush-cli/issues/103)) ([983fb77](https://github.com/alrayyes/hush-hush-cli/commit/983fb7770058283afcb68c5919ddfdf6e1d08184))

## [1.6.0](https://github.com/alrayyes/hush-hush-cli/compare/v1.5.4...v1.6.0) (2026-09-18)


### Features

* add a list command to enumerate stored secrets ([#97](https://github.com/alrayyes/hush-hush-cli/issues/97)) ([5b97809](https://github.com/alrayyes/hush-hush-cli/commit/5b97809e110d8d321fe6619ea68f2205505d2616))

## [1.5.4](https://github.com/alrayyes/hush-hush-cli/compare/v1.5.3...v1.5.4) (2026-09-14)


### Bug Fixes

* **nix:** update flake.lock ([#90](https://github.com/alrayyes/hush-hush-cli/issues/90)) ([b46273a](https://github.com/alrayyes/hush-hush-cli/commit/b46273a9bbf0133be8296f8d08bc703c3a52544e))

## [1.5.3](https://github.com/alrayyes/hush-hush-cli/compare/v1.5.2...v1.5.3) (2026-09-14)


### Bug Fixes

* **deps:** bump the go-dependencies group with 2 updates ([#86](https://github.com/alrayyes/hush-hush-cli/issues/86)) ([5251386](https://github.com/alrayyes/hush-hush-cli/commit/5251386f7be91ae200c61d523f9c5eda35ececce))

## [1.5.2](https://github.com/alrayyes/hush-hush-cli/compare/v1.5.1...v1.5.2) (2026-09-12)


### Bug Fixes

* **packaging:** bump AUR PKGBUILD to v1.5.1 ([#79](https://github.com/alrayyes/hush-hush-cli/issues/79)) ([aa43908](https://github.com/alrayyes/hush-hush-cli/commit/aa43908ad0ed1d1c0acd21b695ac201d97acae7f))

## [1.5.1](https://github.com/alrayyes/hush-hush-cli/compare/v1.5.0...v1.5.1) (2026-09-12)


### Bug Fixes

* **packaging:** stop AUR PKGBUILD from drifting off the latest release ([#77](https://github.com/alrayyes/hush-hush-cli/issues/77)) ([f47b7be](https://github.com/alrayyes/hush-hush-cli/commit/f47b7bebdb1eb2b502b68fa5b39f352c152ee11c))

## [1.5.0](https://github.com/alrayyes/hush-hush-cli/compare/v1.4.7...v1.5.0) (2026-09-12)


### Features

* **cli:** prompt for connection config interactively during init ([#74](https://github.com/alrayyes/hush-hush-cli/issues/74)) ([c7853cf](https://github.com/alrayyes/hush-hush-cli/commit/c7853cf3b088bc5319d85c5648277e90c58e53df))

## [1.4.7](https://github.com/alrayyes/hush-hush-cli/compare/v1.4.6...v1.4.7) (2026-09-11)


### Bug Fixes

* **deps:** bump moby/go-archive and x/crypto to patched versions ([#68](https://github.com/alrayyes/hush-hush-cli/issues/68)) ([4cf709a](https://github.com/alrayyes/hush-hush-cli/commit/4cf709a536dbb9ec4958edd4994dbea275269cd2)), closes [#67](https://github.com/alrayyes/hush-hush-cli/issues/67)

## [1.4.6](https://github.com/alrayyes/hush-hush-cli/compare/v1.4.5...v1.4.6) (2026-09-11)


### Bug Fixes

* **deps:** re-downgrade bun.lock lockfileVersion and guard against drift ([#59](https://github.com/alrayyes/hush-hush-cli/issues/59)) ([6bc7581](https://github.com/alrayyes/hush-hush-cli/commit/6bc758191f5c340958631569ae3b9812b9c216e4))

## [1.4.5](https://github.com/alrayyes/hush-hush-cli/compare/v1.4.4...v1.4.5) (2026-09-11)


### Bug Fixes

* **deps:** bump github.com/alrayyes/hush-hush-go ([#50](https://github.com/alrayyes/hush-hush-cli/issues/50)) ([3ea8053](https://github.com/alrayyes/hush-hush-cli/commit/3ea8053a94613885d85d9ff1e9c0153c6a57aaf1))

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
