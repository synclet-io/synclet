# Changelog

## [1.0.0](https://github.com/synclet-io/synclet/compare/synclet-helm-v0.11.0...synclet-helm-v1.0.0) (2026-10-29)


### ⚠ BREAKING CHANGES

* **executor:** executors require an owner ID; chart sets one per replica and adds PDBs
* **auth:** verified-email OIDC linking, browser-bound OIDC state, registration off by default, shared rate limits in Postgres
* **pipeline:** connector pods run only under the instance-wide service account

### Features

* **auth:** verified-email OIDC linking, browser-bound OIDC state, registration off by default, shared rate limits in Postgres ([9409d68](https://github.com/synclet-io/synclet/commit/9409d68ad523125630ea621281dfa5673726cf1a))
* **pipeline:** connector pods run only under the instance-wide service account ([cc3abf7](https://github.com/synclet-io/synclet/commit/cc3abf7f9ef8cc3e223e492b49562206009bd3af))


### Bug Fixes

* **executor:** executors require an owner ID; chart sets one per replica and adds PDBs ([714fb8f](https://github.com/synclet-io/synclet/commit/714fb8f6cacf33af0f71ff680b2275674ff7887d))

## [0.11.0](https://github.com/synclet-io/synclet/compare/synclet-helm-v0.10.0...synclet-helm-v0.11.0) (2026-10-21)


### Features

* **helm:** mcp.exposure and mcp.paths replace mcp.ingress.mode and paths ([be1b08a](https://github.com/synclet-io/synclet/commit/be1b08a8d86b932c9554d67ad2716a4d7b9e27bf))

## [0.10.0](https://github.com/synclet-io/synclet/compare/synclet-helm-v0.9.0...synclet-helm-v0.10.0) (2026-10-21)


### Features

* **helm:** MCP HTTPRoute twin for the Gateway API in separate mode ([d7c8094](https://github.com/synclet-io/synclet/commit/d7c8094150cfc3c3837a140534b6a9ba5422bf24))

## [0.9.0](https://github.com/synclet-io/synclet/compare/synclet-helm-v0.8.0...synclet-helm-v0.9.0) (2026-10-21)


### Features

* **auth:** OAuth for MCP clients ([e8c994a](https://github.com/synclet-io/synclet/commit/e8c994a30e7606976bf5c0af52010f39f26f0522))
* **mcp:** MCP server for AI clients ([244f3f6](https://github.com/synclet-io/synclet/commit/244f3f62f556888ee99d5e8a0a32248c4fa83180))

## [0.8.0](https://github.com/synclet-io/synclet/compare/synclet-helm-v0.7.0...synclet-helm-v0.8.0) (2026-10-21)


### Features

* **auth:** set default workspace role from OIDC claim ([cb19728](https://github.com/synclet-io/synclet/commit/cb19728c299a6a5afd3d87475536b83d7b934b5b))

## [0.7.0](https://github.com/synclet-io/synclet/compare/synclet-helm-v0.6.0...synclet-helm-v0.7.0) (2026-10-20)


### Features

* **helm:** request and backend timeouts on the HTTPRoute ([b0fd6b4](https://github.com/synclet-io/synclet/commit/b0fd6b4e74f6bd4d5f72e9120506a396f755d30a))

## [0.6.0](https://github.com/synclet-io/synclet/compare/synclet-helm-v0.5.0...synclet-helm-v0.6.0) (2026-10-20)


### Features

* **helm:** Artifact Hub metadata, chart README and per-release change notes ([2cdea6d](https://github.com/synclet-io/synclet/commit/2cdea6da65d8e01ba72938304a8e2b3e26218614))

## [0.5.0](https://github.com/synclet-io/synclet/compare/synclet-helm-v0.4.0...synclet-helm-v0.5.0) (2026-10-20)


### Features

* **helm:** executor metrics Service and ServiceMonitor ([8a6449d](https://github.com/synclet-io/synclet/commit/8a6449d3dd7ce1e1035725e94f1356a5436ebd17))
* **helm:** full CORS config and extraEnv from values ([69095a6](https://github.com/synclet-io/synclet/commit/69095a6d2a165c10ac291951aa96e52976574e70))


### Bug Fixes

* **helm:** legacy jobs and executor probes, reuse-values defaults match values.yaml, 0.4.0 values ([5749ab6](https://github.com/synclet-io/synclet/commit/5749ab63c213529b8cf95511eff3fb811b025d48))
* **helm:** reject a serviceMonitor port the Service lacks, document annotations ([4395aab](https://github.com/synclet-io/synclet/commit/4395aab1be0c47a08a21ccf959af8ca05bdc48e5))
* **helm:** templates render over values reused from older releases ([f218574](https://github.com/synclet-io/synclet/commit/f218574d6d9ca87ae964909ec9e1d8264b96b3f2))
* **notify:** default empty SMTP TLS mode to implicit, expose it in the chart ([2e6a79b](https://github.com/synclet-io/synclet/commit/2e6a79bae0cce8f4fdb2cc2ef3cf60afd7901ae1))
* **notify:** default SMTP port to 465 to match implicit TLS ([baf2723](https://github.com/synclet-io/synclet/commit/baf27231da56c294a8f8f61c99404cbcc1b51bce))

## [0.4.0](https://github.com/synclet-io/synclet/compare/synclet-helm-v0.3.0...synclet-helm-v0.4.0) (2026-10-19)


### Features

* **executor:** persistent /local root on the host for file-writing destinations ([39b87f8](https://github.com/synclet-io/synclet/commit/39b87f8f46b195b673086e1c7f2fa8ff8d1f8efc))
* **helm:** executor pods start in parallel ([ffcac0c](https://github.com/synclet-io/synclet/commit/ffcac0c2ccfd677ca1e77670bea66f3402d4e221))
* **helm:** executor runs as a StatefulSet with a headless Service and its own ConfigMap ([d5916ae](https://github.com/synclet-io/synclet/commit/d5916ae5586ee95e7f44506c179d0fab888d05b2))


### Bug Fixes

* **executor:** connector pods get no Kubernetes API token unless a service account is chosen ([5c0c072](https://github.com/synclet-io/synclet/commit/5c0c0721ee7498bb57f03ec81922e55cb4b0959b))

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
