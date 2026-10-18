# Changelog

## [0.3.0](https://github.com/synclet-io/synclet/compare/synclet-helm-v0.2.0...synclet-helm-v0.3.0) (2026-10-18)


### Features

* **executor:** owner label scopes Docker orphan cleanup to one executor ([c2d3a26](https://github.com/synclet-io/synclet/commit/c2d3a263cf3e06296e661a6323011d715087da69))

## [0.2.0](https://github.com/synclet-io/synclet/compare/synclet-helm-v0.1.3...synclet-helm-v0.2.0) (2026-10-17)


### Features

* **app:** validate secret length and format at startup (F-28) ([b13af3a](https://github.com/synclet-io/synclet/commit/b13af3ae85ea9e9a8bf7d6543c7a31b8704ff85f))
* **helm:** configurable server probe paths and ports (F-27) ([0a32604](https://github.com/synclet-io/synclet/commit/0a326040b67f19cd6a46710f7cf3736daa4915cd))
* **helm:** Gateway API HTTPRoute as an alternative or companion to Ingress ([9e88faa](https://github.com/synclet-io/synclet/commit/9e88faae632a625a22f8cdfde49991f1b8853cdd))
* **helm:** optional bundled Garage for job log archives ([fbfd05a](https://github.com/synclet-io/synclet/commit/fbfd05aa3fec69b6b09be9b96829d29f153503e5))
* **helm:** separate executor and connector service accounts (F-16) ([ef754e5](https://github.com/synclet-io/synclet/commit/ef754e5eed154f65b4868127334eadfad45a0931))
* **pipeline:** archive finished job logs to filesystem, S3 or GCS ([4a7bae5](https://github.com/synclet-io/synclet/commit/4a7bae5983bfffd985aabec7d25484f87b827512))


### Bug Fixes

* **helm:** wait for Postgres and migrate before jobs and executor start (F-21) ([24b6dea](https://github.com/synclet-io/synclet/commit/24b6dea905b52ea83b122901fb04b1a054cae40e))
* **ops:** hashed invite tokens, password rules, codegen drift CI, versioned chart ([59d8e84](https://github.com/synclet-io/synclet/commit/59d8e844d8b597610d676913a64f292e520ce2ed))
* **ops:** readiness probes, migration lock, config and docs match the code ([6103d53](https://github.com/synclet-io/synclet/commit/6103d53324292c37f2aa27307609327bb22500d8))

## [0.1.3](https://github.com/synclet-io/synclet/compare/synclet-helm-v0.1.2...synclet-helm-v0.1.3) (2026-04-01)


### Dependencies

* **chart:** bump synclet version to 0.3.2 ([61c1b1b](https://github.com/synclet-io/synclet/commit/61c1b1b1475f03d844656ced456411c4b9548b22))

## [0.1.2](https://github.com/synclet-io/synclet/compare/synclet-helm-v0.1.1...synclet-helm-v0.1.2) (2026-04-01)


### Bug Fixes

* chart appVersion ([99a2206](https://github.com/synclet-io/synclet/commit/99a2206d4df7174fdfd5107f92f8609304c06324))

## [0.1.1](https://github.com/synclet-io/synclet/compare/synclet-helm-v0.1.0...synclet-helm-v0.1.1) (2026-04-01)


### Bug Fixes

* **ci, chart:** Fix golangci-lint errors, trigger chart release PR ([50978a3](https://github.com/synclet-io/synclet/commit/50978a3b2c250a872187c8164107438fcaa9386d))
