# Changelog

## [0.6.0](https://github.com/synclet-io/synclet/compare/synclet-v0.5.1...synclet-v0.6.0) (2026-10-18)


### Features

* **compose:** one-command quickstart on the published image ([d854386](https://github.com/synclet-io/synclet/commit/d854386d2eaf291d26a9135327eb9c3e46abe803))
* **connectors:** drop connectors/, e2e test connector on synclet-go-sdk v0.3.0 ([a48c70a](https://github.com/synclet-io/synclet/commit/a48c70a9d26790d17aba3df58c27027804052d4b))
* **executor:** executors run their own fx graphs without the database ([5582a83](https://github.com/synclet-io/synclet/commit/5582a83979aacdfafd3e348bce900a2a3c05b65e))
* **executor:** owner label scopes Docker orphan cleanup to one executor ([c2d3a26](https://github.com/synclet-io/synclet/commit/c2d3a263cf3e06296e661a6323011d715087da69))
* **pipeline:** default Synclet registry is served from synclet-io/connectors ([5666e59](https://github.com/synclet-io/synclet/commit/5666e59bde330c3387cce2577ae790db1feb8fc1))


### Bug Fixes

* **ci:** pin golangci-lint, faster test lock retry, default-secret warning ([1fde6c2](https://github.com/synclet-io/synclet/commit/1fde6c25fd78ea0bb6261b70b7589ff2a92eb912))
* **e2e:** test connector links to synclet-io/synclet, CI runs its tests ([13e3d92](https://github.com/synclet-io/synclet/commit/13e3d9201a123d8764c3e71f9acb2a79ba63a652))
* **executor:** drain the attach stream after the container exits ([d424858](https://github.com/synclet-io/synclet/commit/d4248584b5a39be3fa2e0dceb524a05d5624327d))
* **pipeline:** save state under source stream names, map rewritten names back on read ([bde8d68](https://github.com/synclet-io/synclet/commit/bde8d68176fec749bdeae2953da8a5e97a201239))
* **pipeline:** STREAM save after GLOBAL or LEGACY state starts a fresh STREAM-only array ([d15879d](https://github.com/synclet-io/synclet/commit/d15879d8824b5fea5bf6d8e21bf141f081a2760f))
* **release:** archives fail on missing files, sha256sum docs, task test:db ([550923d](https://github.com/synclet-io/synclet/commit/550923d9797930570cbfdf6c6e0761596eb1bd94)), closes [#26](https://github.com/synclet-io/synclet/issues/26) [#27](https://github.com/synclet-io/synclet/issues/27) [#28](https://github.com/synclet-io/synclet/issues/28)

## [0.5.1](https://github.com/synclet-io/synclet/compare/synclet-v0.5.0...synclet-v0.5.1) (2026-10-17)


### Bug Fixes

* **ci:** serialize test-package migrations and close the golangci findings ([cd4df51](https://github.com/synclet-io/synclet/commit/cd4df51b65d70d1e2b7167611fd7dc30b974c1ab))
* **coordinator:** FIFOs openable by any user, fail fast when a connector exits early ([d301ddf](https://github.com/synclet-io/synclet/commit/d301ddf3470b2f06d06cec9860c54789e7163bd6))
* **helm:** preallocate init container names in chart test ([c519f89](https://github.com/synclet-io/synclet/commit/c519f8917a375e2023805524e3f4ea5a1586a26b))
* **pipeline:** catch-up migration for objects missing on old databases ([0e4afc2](https://github.com/synclet-io/synclet/commit/0e4afc22ac9e0d4529c513204d3151cd6501cc43))
* **release:** publish the four tar.gz archives and SHA256SUMS the matrix builds ([00b6c3f](https://github.com/synclet-io/synclet/commit/00b6c3fca376b5740b5ded2427b4a56d12bcd40d))

## [0.5.0](https://github.com/synclet-io/synclet/compare/synclet-v0.4.0...synclet-v0.5.0) (2026-10-17)


### Features

* **app:** validate secret length and format at startup (F-28) ([b13af3a](https://github.com/synclet-io/synclet/commit/b13af3ae85ea9e9a8bf7d6543c7a31b8704ff85f))
* **audit:** add Audit Log module skeleton (table + RPC + use cases) ([6625185](https://github.com/synclet-io/synclet/commit/6625185d4da76ed8169b4af5bd690ca768e1dea8))
* **audit:** add Audit log page under Settings (admin-only) ([a385887](https://github.com/synclet-io/synclet/commit/a385887e5566d2e437aad897bb878b7b9da9228e))
* **audit:** audit stream state updates and workspace config imports ([da22134](https://github.com/synclet-io/synclet/commit/da22134e17b294878beaf21418751c8780278a02))
* **audit:** instrument remaining pipeline, notify and workspace mutations ([a68cc54](https://github.com/synclet-io/synclet/commit/a68cc54526fef4c7f1b50f2040f9c1b0f1785989))
* **audit:** label events whose actor was deleted (F-30) ([248407f](https://github.com/synclet-io/synclet/commit/248407fad3f2a0a39b246955674b5eb318dba7e9))
* **audit:** persist audit events through the tx outbox ([7119d30](https://github.com/synclet-io/synclet/commit/7119d3084e26be38a33b9eeb016404ed49371fa0))
* **audit:** retention job with per-workspace setting ([409b7ee](https://github.com/synclet-io/synclet/commit/409b7eeeac26096f4caa5eb15253067d2b5ff0c9))
* **auditutil:** pure-Go redaction + diff + truncation helpers for the audit module ([ab7e3e2](https://github.com/synclet-io/synclet/commit/ab7e3e203fa9496fe2a92279dbd0b51a8b45e4ae))
* **auth:** record audit events for login, logout, API keys and profile changes ([f7ffb80](https://github.com/synclet-io/synclet/commit/f7ffb80312b62dfaede87ce7590caf4659b9335a))
* **auth:** return workspace memberships from Login and Register (F-22) ([98f2248](https://github.com/synclet-io/synclet/commit/98f224811563a94113616fe8faf7b370a8b9027d))
* **auth:** return workspaces from GetCurrentUser ([b85e78b](https://github.com/synclet-io/synclet/commit/b85e78b75079719e5172dd5127863a601dbb75e9))
* **auth:** self-service password reset with operator fallback ([4a0dc90](https://github.com/synclet-io/synclet/commit/4a0dc90a7ff38138e879dfac196cbf75b790c058))
* **connections:** add Duplicate, multi-select, and Pause/Resume all ([6fda367](https://github.com/synclet-io/synclet/commit/6fda3678bd793a8a6b36738f0917429b1261a7d6))
* **connections:** source and destination columns and filters ([d82f710](https://github.com/synclet-io/synclet/commit/d82f7104de26e99939086fabb720f779091e5acd))
* **docker:** batch job log appends in the Docker executor (F-20) ([0dfae5d](https://github.com/synclet-io/synclet/commit/0dfae5d398af2eaaf5ff7bdbe3f8747169c475a9))
* **events:** retry with backoff and dead-letter for bus handlers ([724c6d7](https://github.com/synclet-io/synclet/commit/724c6d7c2d63c45cdbe2e87aff6a519e94481476))
* **front:** job timeline with per-attempt logs, search keyboard nav, UI polish ([0955106](https://github.com/synclet-io/synclet/commit/095510686240654164469e72b40cadfdfa072162))
* **helm:** configurable server probe paths and ports (F-27) ([0a32604](https://github.com/synclet-io/synclet/commit/0a326040b67f19cd6a46710f7cf3736daa4915cd))
* **helm:** Gateway API HTTPRoute as an alternative or companion to Ingress ([9e88faa](https://github.com/synclet-io/synclet/commit/9e88faae632a625a22f8cdfde49991f1b8853cdd))
* **helm:** optional bundled Garage for job log archives ([fbfd05a](https://github.com/synclet-io/synclet/commit/fbfd05aa3fec69b6b09be9b96829d29f153503e5))
* **helm:** separate executor and connector service accounts (F-16) ([ef754e5](https://github.com/synclet-io/synclet/commit/ef754e5eed154f65b4868127334eadfad45a0931))
* **jobs:** workspace-wide jobs page with server-side paging and filters ([7efa9c9](https://github.com/synclet-io/synclet/commit/7efa9c9d63c0e4b3d71d04ce9eac833ca75847e6))
* **k8s:** apply executor default CPU and memory to sync pods (F-13) ([4b7ce30](https://github.com/synclet-io/synclet/commit/4b7ce3025c5a3613be029e6446ecb53d8013d700))
* **notify:** instrument channel + notification rule mutations with audit events ([44d7bc7](https://github.com/synclet-io/synclet/commit/44d7bc74f4446bcaafb5e33240a70ec7d87f6703))
* **notify:** instrument webhook mutations with audit events ([5542b65](https://github.com/synclet-io/synclet/commit/5542b6558f07737d54f1a304dc0d7fedcdc13714))
* **notify:** SMTP TLS mode and a Mailpit-backed password reset e2e test (F-31) ([debbc85](https://github.com/synclet-io/synclet/commit/debbc8582374ff1675380d475121e344e19d1ea1))
* **pipeline:** archive finished job logs to filesystem, S3 or GCS ([4a7bae5](https://github.com/synclet-io/synclet/commit/4a7bae5983bfffd985aabec7d25484f87b827512))
* **pipeline:** audit source and destination config with secrets redacted ([6fd3aee](https://github.com/synclet-io/synclet/commit/6fd3aeef3affd501a369baa1a460ffc04ec656c7))
* **pipeline:** docker network and volume mounts in runtime config ([d9f1bb7](https://github.com/synclet-io/synclet/commit/d9f1bb7d07a981852ee2e9d535852f94f8412cbf))
* **pipeline:** instrument connection mutations with audit events ([cb32ae8](https://github.com/synclet-io/synclet/commit/cb32ae8f83d41917842891aa81d6702d082814a4))
* **pipeline:** instrument source + destination mutations with audit events ([aa5e8f7](https://github.com/synclet-io/synclet/commit/aa5e8f7ef5d542d5b2cca640e86e6f669f990224))
* **pipeline:** metrics for job log archiving (F-18) ([c6bdb0c](https://github.com/synclet-io/synclet/commit/c6bdb0c70150b8d8750593437a0caf990c0b754e))
* **pipeline:** publish job events via tx outbox, deliver webhooks from notify handler ([b71e60f](https://github.com/synclet-io/synclet/commit/b71e60ff16eb7b80ef38f3a4af7113c3141553b0))
* **pipeline:** purge workspace data on delete, harden scheduler and tasks ([c823d77](https://github.com/synclet-io/synclet/commit/c823d7721d5b6f437ee9d8d71183a729109f0907))
* **search:** add Cmd+K global search palette ([88cbe3f](https://github.com/synclet-io/synclet/commit/88cbe3fc6c62963a50e0c58ce3351f651e697370))
* **settings:** show runtime defaults ([09f4212](https://github.com/synclet-io/synclet/commit/09f42124e15d597b4953bc8d77c0189da3a3363c))
* **topology:** add Topology view for source→connection→destination DAG ([bae146b](https://github.com/synclet-io/synclet/commit/bae146b1bf84d3c495398c5ec7be071f083d996c))
* **ui:** drop starting from the job status filter options (F-33) ([350f441](https://github.com/synclet-io/synclet/commit/350f441dd42f8a5d1fdfdd4014fc4a0e909551fd))
* **ui:** SCheckbox and STextarea, sized skeletons, runtime config validation ([50cef04](https://github.com/synclet-io/synclet/commit/50cef0467633c36348032ba8a14c176b60b65e24))
* **ui:** searchable source and destination filters ([81cea60](https://github.com/synclet-io/synclet/commit/81cea6066d8017b98282605fed6b3e1f6ec047b4))
* **workspace:** instrument member role + remove with audit events ([33609c8](https://github.com/synclet-io/synclet/commit/33609c855372944c343028bee82c40965b9de78f))
* **workspace:** instrument workspace update + invite creation with audit events ([18fc690](https://github.com/synclet-io/synclet/commit/18fc6903683e1f6fdf1c8392212ade6318c78b38))
* **workspace:** show member identities, shareable invite links without SMTP ([b24f96e](https://github.com/synclet-io/synclet/commit/b24f96ea36c8333385348da966722225f75722cc))


### Bug Fixes

* **api:** imports go through use cases, accurate error codes, safe deletes ([2cb3d99](https://github.com/synclet-io/synclet/commit/2cb3d99317031be1bbcbee0b3362ad5e000fe435))
* **config:** fall back to defaults for empty env values ([9268867](https://github.com/synclet-io/synclet/commit/9268867967e3068621fd613163f75bdc1cde51dd))
* **executor:** consumers, auth and cancel in distributed mode ([2ed2cc6](https://github.com/synclet-io/synclet/commit/2ed2cc62092a71202d1a6ae7e77a2deb10d5335c))
* **executor:** redact Docker connector task errors ([e9afbbf](https://github.com/synclet-io/synclet/commit/e9afbbfe60505bf6f71098943b3c89f53550e0f7))
* **executor:** report failed Docker syncs instead of treating them as cancelled ([9aa2943](https://github.com/synclet-io/synclet/commit/9aa2943ea97fd7992d726b84f1a6008324d7f46c))
* **executor:** report running status once a sync starts ([72401cb](https://github.com/synclet-io/synclet/commit/72401cb10728ffcade1f21e151d3b484d1f57b15))
* **front:** connection pages follow route changes, wizard cannot duplicate ([a6671ae](https://github.com/synclet-io/synclet/commit/a6671ae12ab468acdafc177d3a113b7b18cb2858))
* **front:** invalidate every mutation, async schema discovery, guided setup pages ([88c340a](https://github.com/synclet-io/synclet/commit/88c340acb34eba44929cdc7726c8174a3a01cb29))
* **front:** poll active jobs, keep stream config on catalog refresh ([ebd592e](https://github.com/synclet-io/synclet/commit/ebd592e36b48ebeadfc45e678c63f86687b03ce7))
* **front:** readable errors, in-dialog feedback, paged and filtered audit log ([7961c84](https://github.com/synclet-io/synclet/commit/7961c84b18cb7f1607973a2e2cb81429251910e2))
* **front:** settings, dashboard and entity detail polish ([773d052](https://github.com/synclet-io/synclet/commit/773d052a6e8fb70fd17daf97ee053e63db67e2c1))
* **front:** single-flight token refresh, keep redirects and invites through login ([1ea1423](https://github.com/synclet-io/synclet/commit/1ea142343eb4d9b47027e99d1c5b809a0e746dde))
* **front:** unknown enum handling, safe theme storage, fewer list fetches ([dc8bbc4](https://github.com/synclet-io/synclet/commit/dc8bbc430cfe762c9c6be8f021d5572c91ae53cf))
* **helm:** wait for Postgres and migrate before jobs and executor start (F-21) ([24b6dea](https://github.com/synclet-io/synclet/commit/24b6dea905b52ea83b122901fb04b1a054cae40e))
* **ops:** guard migrate down, refuse missing secrets, block in-use deletes ([619765f](https://github.com/synclet-io/synclet/commit/619765f7d79aa4590f6b1221a9fad9b5843ea69c))
* **ops:** hashed invite tokens, password rules, codegen drift CI, versioned chart ([59d8e84](https://github.com/synclet-io/synclet/commit/59d8e844d8b597610d676913a64f292e520ce2ed))
* **ops:** readiness probes, migration lock, config and docs match the code ([6103d53](https://github.com/synclet-io/synclet/commit/6103d53324292c37f2aa27307609327bb22500d8))
* **pipeline:** no archive object outlives its job (F-19) ([23a206d](https://github.com/synclet-io/synclet/commit/23a206d3e0c1960d886274bda5f50c25010c74eb))
* **pipeline:** read top-level breakingChanges from the connector registry (F-14) ([455b3c8](https://github.com/synclet-io/synclet/commit/455b3c8443fc232d819c0e468522c13492b78176))
* **pipeline:** terminal job states win, masked secrets survive edits ([818b71b](https://github.com/synclet-io/synclet/commit/818b71b6377686e59679cd608025048866053c3b))
* **pipeline:** upsert stream generations so repeated syncs succeed (F-26) ([c844424](https://github.com/synclet-io/synclet/commit/c84442480a507dde8f9108180d5cbf7272f1feec))
* **security:** login lockout, per-procedure rate limits, secret redaction ([ed61760](https://github.com/synclet-io/synclet/commit/ed61760ef5058d4858a69284aee9248b3395e094))
* **security:** spec-aware audit redaction, token refresh race, webhook secret rotation ([c7bd5d2](https://github.com/synclet-io/synclet/commit/c7bd5d214143c5e0faafc0520b06187632bddda9))
* **ui:** accessible form fields, dialogs, menus, tabs, toasts and buttons ([ecaf9ab](https://github.com/synclet-io/synclet/commit/ecaf9ab66c12696aaed85083988d658b7e65e076))
* **ui:** align SSelect chevron with a consistent inset ([44b3000](https://github.com/synclet-io/synclet/commit/44b3000c976b50746360e89ed6963461fc557aff))
* **ui:** charts read theme colours and re-render on theme change ([79db0b4](https://github.com/synclet-io/synclet/commit/79db0b4d66fdacea8b00e98d094a0eca4be47d6c))
* **ui:** consistent headings, buttons, badges and trend indicators across pages ([95e16d1](https://github.com/synclet-io/synclet/commit/95e16d13cb286899980b7cbddc3f0ffc1e738fdd))
* **ui:** contrast-safe theme tokens, no dark-mode flash ([0d56cb3](https://github.com/synclet-io/synclet/commit/0d56cb363d8ad9142f4465459da3915b9bd8f740))
* **ui:** keyboard reach for streams, topology, search and mobile nav ([a9fb28d](https://github.com/synclet-io/synclet/commit/a9fb28d2e11cb2ad8433c1035cd004c8b3807f01))
* **ui:** raise light-theme muted text contrast on hover (F-08) ([77e07e1](https://github.com/synclet-io/synclet/commit/77e07e1b5dc0b7d4b1ec1e27c604ddfdc046452d))
* **ui:** solid tokens on filled controls, badge swap and alert dismiss label ([485f183](https://github.com/synclet-io/synclet/commit/485f183ce3a37b9d69b6f264146f202044745032))

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
