# Changelog

## [0.5.0](https://github.com/synclet-io/synclet/compare/synclet-v0.4.0...synclet-v0.5.0) (2026-06-03)


### Features

* **audit:** add Audit Log module skeleton (table + RPC + use cases) ([6625185](https://github.com/synclet-io/synclet/commit/6625185d4da76ed8169b4af5bd690ca768e1dea8))
* **audit:** add Audit log page under Settings (admin-only) ([a385887](https://github.com/synclet-io/synclet/commit/a385887e5566d2e437aad897bb878b7b9da9228e))
* **auditutil:** pure-Go redaction + diff + truncation helpers for the audit module ([ab7e3e2](https://github.com/synclet-io/synclet/commit/ab7e3e203fa9496fe2a92279dbd0b51a8b45e4ae))
* **connections:** add Duplicate, multi-select, and Pause/Resume all ([6fda367](https://github.com/synclet-io/synclet/commit/6fda3678bd793a8a6b36738f0917429b1261a7d6))
* **notify:** instrument channel + notification rule mutations with audit events ([44d7bc7](https://github.com/synclet-io/synclet/commit/44d7bc74f4446bcaafb5e33240a70ec7d87f6703))
* **notify:** instrument webhook mutations with audit events ([5542b65](https://github.com/synclet-io/synclet/commit/5542b6558f07737d54f1a304dc0d7fedcdc13714))
* **pipeline:** instrument connection mutations with audit events ([cb32ae8](https://github.com/synclet-io/synclet/commit/cb32ae8f83d41917842891aa81d6702d082814a4))
* **pipeline:** instrument source + destination mutations with audit events ([aa5e8f7](https://github.com/synclet-io/synclet/commit/aa5e8f7ef5d542d5b2cca640e86e6f669f990224))
* **search:** add Cmd+K global search palette ([88cbe3f](https://github.com/synclet-io/synclet/commit/88cbe3fc6c62963a50e0c58ce3351f651e697370))
* **topology:** add Topology view for source→connection→destination DAG ([bae146b](https://github.com/synclet-io/synclet/commit/bae146b1bf84d3c495398c5ec7be071f083d996c))
* **workspace:** instrument member role + remove with audit events ([33609c8](https://github.com/synclet-io/synclet/commit/33609c855372944c343028bee82c40965b9de78f))
* **workspace:** instrument workspace update + invite creation with audit events ([18fc690](https://github.com/synclet-io/synclet/commit/18fc6903683e1f6fdf1c8392212ade6318c78b38))

## [0.4.0](https://github.com/synclet-io/synclet/compare/synclet-v0.3.2...synclet-v0.4.0) (2026-05-17)


### Features

* **app:** set default environment variables ([bcb1824](https://github.com/synclet-io/synclet/commit/bcb18248b2c7f38d94fad460be75e1f49f357a0d))
* **connection:** surface GetSchemaChanges in the UI ([85fbc8d](https://github.com/synclet-io/synclet/commit/85fbc8dea47bb0e091989daa2ef76137eb8550e9))
* **front:** connector catalog browse page (Phase 1) ([aef778b](https://github.com/synclet-io/synclet/commit/aef778be7ff9e23aca9128b736d3b0a1c72c6d51))
* **front:** guided onboarding checklist on the dashboard ([895b1d9](https://github.com/synclet-io/synclet/commit/895b1d90c665612c648fa848deed64725db82411))
* **webhook:** add management UI and TestWebhook RPC ([6c50c4b](https://github.com/synclet-io/synclet/commit/6c50c4b0b49693269a29775309de76b92b22f904))
* **workspace,pipeline:** seed Airbyte OSS + Synclet registries on workspace creation ([32c93b7](https://github.com/synclet-io/synclet/commit/32c93b7b6ffde79f87c90ff053f92cafdb6526d2))
* **workspace:** add UpdateMemberRole RPC end-to-end ([2305bdd](https://github.com/synclet-io/synclet/commit/2305bdd76ad7a0c1fdba6033e277f8c8c5c80510))
* **workspace:** finish workspace event publishing through the outbox ([570b09d](https://github.com/synclet-io/synclet/commit/570b09d72190b7a18e7e5c6464ea7ca8f97d77af))
* **workspace:** return caller's role from ListWorkspaces ([54eebfa](https://github.com/synclet-io/synclet/commit/54eebfa956a34d63c11d085d5bb7f2190363ce47))


### Bug Fixes

* docs link ([d6d1b4f](https://github.com/synclet-io/synclet/commit/d6d1b4fdbb6061eeb9dab40b05aa693fade50a4f))
* **front:** catalog browse aggregates connectors from all repositories ([946d6f0](https://github.com/synclet-io/synclet/commit/946d6f0aeaf703e5bc92c7ba5ac3326d2ee7d175))
* **front:** replace window.confirm with SConfirmDialog; surface stream-config failures ([ff13762](https://github.com/synclet-io/synclet/commit/ff137623fe7e186940e56dfbcf38d4248a59472c))
* **front:** tab highlight, repo dropdown size, responsive checklist ([15137fe](https://github.com/synclet-io/synclet/commit/15137fe0e938dd4c36988320d1959e375375a3bd))
* **front:** use console.warn instead of console.debug in StreamConfigPage ([ebca853](https://github.com/synclet-io/synclet/commit/ebca8538430db5e120c12578e78f4ec9123b970a))
* **pipeline:** make connector scratch dir host-mountable in Docker-in-Docker ([1f66ed2](https://github.com/synclet-io/synclet/commit/1f66ed2f34ad75e89c405eb313ff64613f81f5b2))
* **security:** allow https: in img-src for connector catalog icons ([cd69da6](https://github.com/synclet-io/synclet/commit/cd69da646b78e1184cb3c97d0cd4b1711b95cce4))
* **synclet-helm:** bump appVersion to match synclet release ([07c7707](https://github.com/synclet-io/synclet/commit/07c7707339a4d127fc8bb8952094895027359d0f))
* **workspace:** emit workspace.created from bootstrap so default registries seed ([3a567cd](https://github.com/synclet-io/synclet/commit/3a567cd3d19eb8a1b93e2a1929f6a0587bc332e6))

## [0.3.2](https://github.com/synclet-io/synclet/compare/synclet-v0.3.1...synclet-v0.3.2) (2026-04-01)


### Bug Fixes

* cross-platform docker images release ([9218f2d](https://github.com/synclet-io/synclet/commit/9218f2d7bb688237ccce9c5da522cb3c4517a438))

## [0.3.1](https://github.com/synclet-io/synclet/compare/synclet-v0.3.0...synclet-v0.3.1) (2026-04-01)


### Bug Fixes

* chart appVersion ([99a2206](https://github.com/synclet-io/synclet/commit/99a2206d4df7174fdfd5107f92f8609304c06324))
* **ci, chart:** Fix golangci-lint errors, trigger chart release PR ([50978a3](https://github.com/synclet-io/synclet/commit/50978a3b2c250a872187c8164107438fcaa9386d))

## [0.3.0](https://github.com/synclet-io/synclet/compare/synclet-v0.2.1...synclet-v0.3.0) (2026-04-01)


### Features

* **connectors:** add meta.json metadata and forward in release workflow ([ce73f9f](https://github.com/synclet-io/synclet/commit/ce73f9f144855d60a11dcf26aa945c0753777531))
* **pipeline:** add connector filter UI with repository dropdown and search ([1fee161](https://github.com/synclet-io/synclet/commit/1fee1610102194e946acf8dc58b446f8b2265990))
* **pipeline:** add filter fields to ListManagedConnectorsRequest proto ([caf6eca](https://github.com/synclet-io/synclet/commit/caf6eca7929a5a5cab0facc711c999809307de5b))
* **pipeline:** add repository and search filters to ListManagedConnectors ([11605c5](https://github.com/synclet-io/synclet/commit/11605c58cfcdbf405a2f5d8dd284867f70061257))
* **pipeline:** filter managed connectors by repository and name ([236729d](https://github.com/synclet-io/synclet/commit/236729d331bcbebaed291a2a1017d908cecb2bc0))


### Bug Fixes

* **auth:** use citext for email ([1c9c379](https://github.com/synclet-io/synclet/commit/1c9c3795048b0b010f72fc1e1a40d1a0489da3f9))
* connectors name ([a9f05f1](https://github.com/synclet-io/synclet/commit/a9f05f18ff31d2f3a8f411734dec37a69bd31c65))
* **pipeline:** add pre-delete validation for managed connectors ([12035a8](https://github.com/synclet-io/synclet/commit/12035a831cabedd33b382e1c85a8d1ee38c63ee6))
* **pipeline:** add pre-delete validation for managed connectors ([12035a8](https://github.com/synclet-io/synclet/commit/12035a831cabedd33b382e1c85a8d1ee38c63ee6))
* **pipeline:** add pre-delete validation for managed connectors ([3074713](https://github.com/synclet-io/synclet/commit/3074713f958a9ec168aa6afc2090333de4e8542a))
* return domain errors from application layer instead of errors.New ([45e355f](https://github.com/synclet-io/synclet/commit/45e355fbf132797d53ba5de354d63e1813d5c60b))
* trigger build ([7ed32fe](https://github.com/synclet-io/synclet/commit/7ed32fecc87a0350302a1a7c78217a5a859459f3))
* trigger build ([d527235](https://github.com/synclet-io/synclet/commit/d52723525a2feb333f26caefa30bb63972eb4ae5))
* update registry workflow via app token ([6a4fbac](https://github.com/synclet-io/synclet/commit/6a4fbac46f41e996a37fc0472ab11dffe8253e06))
* use github app to create PRs and Releases ([a653f0e](https://github.com/synclet-io/synclet/commit/a653f0e84fd8d4d884bca34f116e9af49a3b4e7b))

## [0.2.1](https://github.com/synclet-io/synclet/compare/synclet-v0.2.0...synclet-v0.2.1) (2026-04-01)


### Bug Fixes

* **connectors:** remove unused dependencies from Dockerfiles ([7cc6ab4](https://github.com/synclet-io/synclet/commit/7cc6ab437b8e4249da91ad86b60560adb7501c11))

## [0.2.0](https://github.com/synclet-io/synclet/compare/synclet-v0.1.0...synclet-v0.2.0) (2026-04-01)


### Features

* **ci:** modernize CI/CD pipeline for dynamic connector support ([b1f5043](https://github.com/synclet-io/synclet/commit/b1f5043c4059b26938b674507292918237b9c7cd))
