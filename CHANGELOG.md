# Changelog

## [1.4.0](https://github.com/woodleighschool/woodgate/compare/1.3.0...1.4.0) (2026-08-22)


### Features

* **container:** update image golang (1.26.6 → 1.27.0) ([#101](https://github.com/woodleighschool/woodgate/issues/101)) ([85d5b1a](https://github.com/woodleighschool/woodgate/commit/85d5b1ab00d8837bfe6a3a3b056991ac368feb62))
* **go:** update module github.com/go-pkgz/auth/v2 (v2.1.6 → v2.2.0) ([#99](https://github.com/woodleighschool/woodgate/issues/99)) ([b888c62](https://github.com/woodleighschool/woodgate/commit/b888c625db0cd0df565106541170b78546df5164))
* **go:** update module github.com/oapi-codegen/runtime (v1.6.0 → v1.7.0) ([#93](https://github.com/woodleighschool/woodgate/issues/93)) ([37bad85](https://github.com/woodleighschool/woodgate/commit/37bad85bd01d0c361d15e1fd9485ad307412f9c4))
* **npm:** update dependency @vitejs/plugin-react (6.0.5 → 6.1.0) ([#106](https://github.com/woodleighschool/woodgate/issues/106)) ([aaa3574](https://github.com/woodleighschool/woodgate/commit/aaa357425f732d6e0d39f265d5851bd782e83fb9))
* **npm:** update dependency pnpm (11.21.0 → 11.22.0) ([#84](https://github.com/woodleighschool/woodgate/issues/84)) ([adcdfb6](https://github.com/woodleighschool/woodgate/commit/adcdfb6bf0a6096f96dbed1a4a4145165624d6cd))


### Bug Fixes

* **go:** update module github.com/go-chi/chi/v5 (v5.3.1 → v5.3.2) ([#107](https://github.com/woodleighschool/woodgate/issues/107)) ([b7ff522](https://github.com/woodleighschool/woodgate/commit/b7ff522e0b2823ee6b19cb54a9cd29e575c2d2b9))
* **lefthook:** allow ignored formatter inputs ([a12ec17](https://github.com/woodleighschool/woodgate/commit/a12ec1720650456e466c36bae006ff27b490a8a1))
* **npm:** update dependency vite (8.2.1 → 8.2.2) ([#103](https://github.com/woodleighschool/woodgate/issues/103)) ([84cd75f](https://github.com/woodleighschool/woodgate/commit/84cd75f1970d5deaba9b2a184a4d856be09b9b77))
* **pnpm:** regenerate mature lockfile ([9a2ba09](https://github.com/woodleighschool/woodgate/commit/9a2ba0901bef28eb2f9bfc25a7ea619b23b6f10b))
* **tooling:** group toolchain updates ([81bdd0f](https://github.com/woodleighschool/woodgate/commit/81bdd0f3c11d8404e15fa771721635fcd3e67f28))


### Documentation

* clarify repository guidance ([0e2096f](https://github.com/woodleighschool/woodgate/commit/0e2096fb5de5eb2d63b36545fc9bcc53047b1a36))
* document companion app releases ([e311e3c](https://github.com/woodleighschool/woodgate/commit/e311e3c4bef017f49d52eea2f2d0579cf29beaa5))


### Tests

* share valid configuration setup ([5973163](https://github.com/woodleighschool/woodgate/commit/597316352f381c7b6a06fc8548892193bed9522d))


### Build System

* **app:** align the companion target ([3df2060](https://github.com/woodleighschool/woodgate/commit/3df206078e64fa22de5de919c13a832c0a051d42))


### Continuous Integration

* **github-action:** Update action home-operations/.github/actions/workflow-lint (v1.0.2 → v1.0.3) ([#102](https://github.com/woodleighschool/woodgate/issues/102)) ([9f532e7](https://github.com/woodleighschool/woodgate/commit/9f532e7c3f40123dcd0fdf6e2385171a08829434))
* **github-action:** Update action jdx/mise-action (v4.2.4 → v4.2.5) ([71d8d73](https://github.com/woodleighschool/woodgate/commit/71d8d737091b8bc7f79b576403622f833a0cfedd))
* split companion app releases ([bbdf488](https://github.com/woodleighschool/woodgate/commit/bbdf4880d367321e9f0a991e57717a5fd4593b0c))
* standardize workflows and add govulncheck ([223344c](https://github.com/woodleighschool/woodgate/commit/223344c5c81afd9e97a4824906d20a7016cec48c))
* sync shared repository tooling ([6114b3d](https://github.com/woodleighschool/woodgate/commit/6114b3db4fcf56935ab1c81e7fe3df51dd0a780b))


### Miscellaneous Chores

* align ignore rules ([8cd84cd](https://github.com/woodleighschool/woodgate/commit/8cd84cdb3e5b4d2001b6e751cacef1decb8ee519))
* align repository conventions ([78f554f](https://github.com/woodleighschool/woodgate/commit/78f554f0367350e783318cfb53bf5049757dac30))
* fix readme ([7626e02](https://github.com/woodleighschool/woodgate/commit/7626e029c8e234f11624302eaca7d4dca3d09c2b))
* **go:** update toolchain to 1.27 ([ca39139](https://github.com/woodleighschool/woodgate/commit/ca3913937f7101acfe1cf1c73c1b4d79401d9b11))
* **main:** release app 1.3.1 ([#108](https://github.com/woodleighschool/woodgate/issues/108)) ([b37d7a7](https://github.com/woodleighschool/woodgate/commit/b37d7a72c75de725131550e6adcdcf00559e3d9e))
* **mise:** update tool golangci-lint (2.13.0 → 2.13.1) ([#104](https://github.com/woodleighschool/woodgate/issues/104)) ([bb3feae](https://github.com/woodleighschool/woodgate/commit/bb3feae344f5793573078c0ad1a5313612d9dc05))
* **mise:** update tool lefthook (2.1.10 → 2.1.11) ([#105](https://github.com/woodleighschool/woodgate/issues/105)) ([f30a5a3](https://github.com/woodleighschool/woodgate/commit/f30a5a34d349590cdc4ba4abff3318160444856e))
* **pnpm:** use default release cooldown ([4b00693](https://github.com/woodleighschool/woodgate/commit/4b00693659dab0024a10d4b7dfb4a9b1267d2c78))
* post bootstrap cleanup ([4fb6176](https://github.com/woodleighschool/woodgate/commit/4fb6176a8ec20f6ae758b3dc85b59586c0f557f2))
* **release-please:** sync configuration ([b49192f](https://github.com/woodleighschool/woodgate/commit/b49192f666fb2719747433392993b93c6e18cfd6))
* remove redundant self-references ([e1ca374](https://github.com/woodleighschool/woodgate/commit/e1ca3748fe79cce8715b01dcaa33f5f0ece25ea2))
* **tooling:** sync shared configuration ([2da92c0](https://github.com/woodleighschool/woodgate/commit/2da92c02e6c7ddf7bc3bc64e13f4c872c46aa408))

## [1.3.0](https://github.com/woodleighschool/woodgate/compare/1.2.1...1.3.0) (2026-08-11)


### Features

* **deps:** update dependency vite (8.1.5 → 8.2.0) ([#79](https://github.com/woodleighschool/woodgate/issues/79)) ([5dcb47f](https://github.com/woodleighschool/woodgate/commit/5dcb47fe1f052b2e47bcc27723d41867075c3f82))


### Bug Fixes

* **auth:** allow anonymous login redirect ([ee6829b](https://github.com/woodleighschool/woodgate/commit/ee6829b387690e94bbcb3c6968c2a44fda2a999d))
* **ci:** disable automatic mise installs ([e381fe2](https://github.com/woodleighschool/woodgate/commit/e381fe292bc92aefb47351e65d296a1dcf635c09))
* **compose:** keep postgres internal ([1e6a8da](https://github.com/woodleighschool/woodgate/commit/1e6a8daed5afe5556f458a10fe9122299550e22c))
* **hooks:** skip pnpm lockfile formatting ([519bd1e](https://github.com/woodleighschool/woodgate/commit/519bd1ed8c4336ada0b159d831ab49fb2b07c0ec))
* **renovate:** wait for complete toolchain groups ([6d1f7ec](https://github.com/woodleighschool/woodgate/commit/6d1f7ec2a16b5942a0891db5847e1b054cf069c0))

## [1.2.1](https://github.com/woodleighschool/woodgate/compare/1.2.0...1.2.1) (2026-07-30)


### Bug Fixes

* **entra:** request user departments ([f355a6f](https://github.com/woodleighschool/woodgate/commit/f355a6f29635e6135466d22e57e49f43e8df06c0))

## [1.2.0](https://github.com/woodleighschool/woodgate/compare/1.1.0...1.2.0) (2026-07-30)


### Features

* **checkins:** add department filtering ([393c38d](https://github.com/woodleighschool/woodgate/commit/393c38d0cae4bc884cb38abaae9ce0eeec73b5a2))


### Bug Fixes

* **checkins:** label directions and fix sorting ([d600797](https://github.com/woodleighschool/woodgate/commit/d600797abdf2c57c7bd721b33303c560ce237b30))
* **ci:** commit generated web API types ([1253898](https://github.com/woodleighschool/woodgate/commit/125389812556de313663313c5baea09343ba52af))
* **ci:** limit Periphery to macOS ([1a3159c](https://github.com/woodleighschool/woodgate/commit/1a3159c34c1147dedd7e865c4e1713a8a5077652))
* **deps:** update dependency @vitejs/plugin-react (6.0.4 → 6.0.5) ([#77](https://github.com/woodleighschool/woodgate/issues/77)) ([dd451ff](https://github.com/woodleighschool/woodgate/commit/dd451ffbd9ea734af9e32ce2c15a6b71784569c6))

## [1.1.0](https://github.com/woodleighschool/woodgate/compare/1.0.1...1.1.0) (2026-07-30)


### Features

* **deps:** update dependency eslint-plugin-import-x (4.16.2 → 4.17.1) ([#36](https://github.com/woodleighschool/woodgate/issues/36)) ([5e1e8aa](https://github.com/woodleighschool/woodgate/commit/5e1e8aa7835a5ed58e49f8a32dda034b98ad931a))
* **deps:** update dependency eslint-plugin-sonarjs (4.0.2 → 4.2.0) ([#37](https://github.com/woodleighschool/woodgate/issues/37)) ([655f5c0](https://github.com/woodleighschool/woodgate/commit/655f5c024288ab7ddbf3c28edec32dbb96fbf7ce))
* **deps:** update dependency globals (17.4.0 → 17.7.0) ([#38](https://github.com/woodleighschool/woodgate/issues/38)) ([fcef385](https://github.com/woodleighschool/woodgate/commit/fcef3853a343524bd059b0b9c9d87bff1a49eef8))
* **deps:** update dependency prettier (3.8.1 → 3.9.5) ([#39](https://github.com/woodleighschool/woodgate/issues/39)) ([c23f9d3](https://github.com/woodleighschool/woodgate/commit/c23f9d3288bbd3b32679be3e6027148ece55e934))
* **deps:** update dependency react-admin (5.14.4 → 5.15.1) ([#40](https://github.com/woodleighschool/woodgate/issues/40)) ([ad037fd](https://github.com/woodleighschool/woodgate/commit/ad037fd40c4506f7186f6662e93ba889381ead76))
* **deps:** update dependency vite (8.0.16 → 8.1.5) ([#42](https://github.com/woodleighschool/woodgate/issues/42)) ([724028e](https://github.com/woodleighschool/woodgate/commit/724028e87be824103ab9b281e0787cbb845a0aad))
* **deps:** update dependency zod (4.3.6 → 4.4.3) ([#43](https://github.com/woodleighschool/woodgate/issues/43)) ([77040b7](https://github.com/woodleighschool/woodgate/commit/77040b7b5de0b8895cc5d6081d5706dde8f09c8a))
* **deps:** update module github.com/go-chi/chi/v5 (v5.2.5 → v5.3.1) ([#44](https://github.com/woodleighschool/woodgate/issues/44)) ([2b026ad](https://github.com/woodleighschool/woodgate/commit/2b026adb1ef772d17cbb00a99ccaaf845ee09a42))
* **deps:** update module github.com/go-pkgz/rest (v1.21.0 → v1.23.1) ([#45](https://github.com/woodleighschool/woodgate/issues/45)) ([f6bafaf](https://github.com/woodleighschool/woodgate/commit/f6bafaf91b6d4bce4dbe5eb6012c9c240db8b426))
* **deps:** update module github.com/jackc/pgx/v5 (v5.9.2 → v5.10.0) ([#59](https://github.com/woodleighschool/woodgate/issues/59)) ([fed61c2](https://github.com/woodleighschool/woodgate/commit/fed61c226d0840fb625184f03f8cbdd30065bb12))
* **deps:** update module github.com/oapi-codegen/oapi-codegen/v2 (v2.6.0 → v2.7.2) ([#46](https://github.com/woodleighschool/woodgate/issues/46)) ([5f6742c](https://github.com/woodleighschool/woodgate/commit/5f6742c7226c867f1b4db2709db08a75ab27b081))
* **deps:** update module github.com/oapi-codegen/runtime (v1.3.0 → v1.6.0) ([#47](https://github.com/woodleighschool/woodgate/issues/47)) ([80bff95](https://github.com/woodleighschool/woodgate/commit/80bff954cc9c0c2563146ccbc0db991d872dd4aa))
* **deps:** update module github.com/sqlc-dev/sqlc (v1.30.0 → v1.31.1) ([#48](https://github.com/woodleighschool/woodgate/issues/48)) ([52b976e](https://github.com/woodleighschool/woodgate/commit/52b976eb25e7761f995f3fb95aaae2ac8112a05a))
* **deps:** update module github.com/woodleighschool/go-entrasync (v0.1.0 → v0.3.0) ([#49](https://github.com/woodleighschool/woodgate/issues/49)) ([03ce7a5](https://github.com/woodleighschool/woodgate/commit/03ce7a57b8fb12c543caf098782284eb16a03df2))
* **deps:** update node.js (v25.7.0 → v25.9.0) ([#50](https://github.com/woodleighschool/woodgate/issues/50)) ([56f75b7](https://github.com/woodleighschool/woodgate/commit/56f75b70f75677350040020995fb395c43c04c16))
* **deps:** update react monorepo ([#31](https://github.com/woodleighschool/woodgate/issues/31)) ([2fc2630](https://github.com/woodleighschool/woodgate/commit/2fc26306140656639b4fb3d54dfe04a32abc3732))


### Bug Fixes

* **ci:** adopt release please ([dbe7d4b](https://github.com/woodleighschool/woodgate/commit/dbe7d4bc1381e3c83afba63d316410df8091906d))
* **deps:** update backend non-major dependencies ([ed0a62f](https://github.com/woodleighschool/woodgate/commit/ed0a62fcb3415bfb3c016b3f14fb7e868e95ddb9))
* **deps:** update backend non-major dependencies ([76ba6cc](https://github.com/woodleighschool/woodgate/commit/76ba6cced7cda6cac3d4b2aadc536c59f457b16e))
* **deps:** update dependency eslint-import-resolver-typescript (4.4.4 → 4.4.5) ([#25](https://github.com/woodleighschool/woodgate/issues/25)) ([ea709cd](https://github.com/woodleighschool/woodgate/commit/ea709cd87d3c591d2d2a63f067743d63712ed867))
* **deps:** update dependency vite (8.0.2 → 8.0.16) [security] ([#32](https://github.com/woodleighschool/woodgate/issues/32)) ([f7982e4](https://github.com/woodleighschool/woodgate/commit/f7982e412734c1c74c3aaf806e3cc990752de30a))
* **deps:** update eslint monorepo (9.39.4 → 9.39.5) ([#26](https://github.com/woodleighschool/woodgate/issues/26)) ([3947637](https://github.com/woodleighschool/woodgate/commit/39476376f272a8594614e85c79a40cc3e4f3c8b1))
* **deps:** update material-ui monorepo (7.3.9 → 7.3.11) ([#28](https://github.com/woodleighschool/woodgate/issues/28)) ([e370d72](https://github.com/woodleighschool/woodgate/commit/e370d7276f960cfe2ca60114d2a3bac1d5337f5e))
* **deps:** update module github.com/caarlos0/env/v11 (v11.4.0 → v11.4.1) ([#29](https://github.com/woodleighschool/woodgate/issues/29)) ([5166887](https://github.com/woodleighschool/woodgate/commit/5166887d6820014ac71a57c88b394b929a4a8910))
* **deps:** update module github.com/go-pkgz/auth/v2 (v2.1.2 → v2.1.5) ([#35](https://github.com/woodleighschool/woodgate/issues/35)) ([30be8ae](https://github.com/woodleighschool/woodgate/commit/30be8ae56f236b24b5cef594fa1c8ea6f3de2746))
* **deps:** update module github.com/jackc/pgx/v5 (v5.9.1 → v5.9.2) [security] ([#34](https://github.com/woodleighschool/woodgate/issues/34)) ([54c5d69](https://github.com/woodleighschool/woodgate/commit/54c5d6991c3931a532275118228570519a7b7de4))
* **deps:** update module github.com/pressly/goose/v3 (v3.27.0 → v3.27.2) ([#30](https://github.com/woodleighschool/woodgate/issues/30)) ([0cc3ba7](https://github.com/woodleighschool/woodgate/commit/0cc3ba7c999f8677d2e1e28611d8fff8571781f0))


### Code Refactoring

* flatten application layout ([5f6bad0](https://github.com/woodleighschool/woodgate/commit/5f6bad0446bb890f29d2b64eb1b13aad2928db0c))
