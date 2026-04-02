# Changelog

## [0.3.3](https://github.com/synclet-io/synclet/compare/synclet-v0.3.2...synclet-v0.3.3) (2026-04-02)


### Bug Fixes

* docs link ([d6d1b4f](https://github.com/synclet-io/synclet/commit/d6d1b4fdbb6061eeb9dab40b05aa693fade50a4f))
* **synclet-helm:** bump appVersion to match synclet release ([07c7707](https://github.com/synclet-io/synclet/commit/07c7707339a4d127fc8bb8952094895027359d0f))

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
