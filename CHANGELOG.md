# Changelog

## [0.4.0](https://github.com/kausys/apikit/compare/v0.3.3...v0.4.0) (2026-03-12)


### Features

* consolidate from 5 modules to 2 (root + cmd) ([#7](https://github.com/kausys/apikit/issues/7)) ([e81b38e](https://github.com/kausys/apikit/commit/e81b38e2de23f3472047e44355ac50f5a11f222f))

## [0.3.3](https://github.com/kausys/apikit/compare/v0.3.2...v0.3.3) (2026-03-12)


### Bug Fixes

* support fully-qualified import paths for custom types ([#4](https://github.com/kausys/apikit/issues/4)) ([fbdfe38](https://github.com/kausys/apikit/commit/fbdfe38b2c392b166d7fc0a6876e1b9e8c56c863))


### Miscellaneous

* add payload sanitization, swagger UI handler, and logging improvements ([#5](https://github.com/kausys/apikit/issues/5)) ([895aeb9](https://github.com/kausys/apikit/commit/895aeb99e964e66bff907e506c291c3080d6cebd))

## [0.3.2](https://github.com/kausys/apikit/compare/v0.3.1...v0.3.2) (2026-03-12)


### Bug Fixes

* disable setup-go cache to avoid tar conflicts ([5bb19d7](https://github.com/kausys/apikit/commit/5bb19d7f8b9cf7be68dfb7d88fa1f58de3a136b0))
* fix CI build for cmd/apikit directory structure ([c155286](https://github.com/kausys/apikit/commit/c15528647c968ebc0aeb272a6f8912aed10da64e))
* use -o temp dir only for cmd module build ([37e0a37](https://github.com/kausys/apikit/commit/37e0a379ac5e200a5e4f3272fb9b77578863dea1))

## [0.3.0](https://github.com/kausys/apikit/compare/v0.2.0...v0.3.0) (2026-03-12)


### Features

* initial release v0.2.0 ([27d280d](https://github.com/kausys/apikit/commit/27d280d94f8429fea31c889971680f274aa6a083))

## [0.3.0](https://github.com/kausys/apikit/compare/v0.2.0...v0.3.0) (2026-03-12)


### Features

* initial release v0.2.0 ([1ff0b33](https://github.com/kausys/apikit/commit/1ff0b338615ded4642dd0bde4967b9a44d50b7ca))


### Bug Fixes

* run go mod tidy in cmd dir for goreleaser ([fcb719b](https://github.com/kausys/apikit/commit/fcb719b7395c3285918c6b8472fbd7ac666b0d59))

## [0.2.0](https://github.com/kausys/apikit/compare/v0.1.2...v0.2.0) (2026-03-12)


### Features

* **scanner:** improve test readability with multi-line string formatting ([a6dd22a](https://github.com/kausys/apikit/commit/a6dd22a3f2361bb14e7f7afcb26c0c3c56aee437))


### Bug Fixes

* **scanner:** fix dupword false positive in TestScanSimpleModel ([e386c69](https://github.com/kausys/apikit/commit/e386c69448695cbeb9eb816e741ef803ff18607c))
* **scanner:** restore missing ID field in TestScanSimpleModel ([d3bf480](https://github.com/kausys/apikit/commit/d3bf480a7e1175c4a4fd695c87cfa987803ce8bd))
* **scanner:** restore missing ID field removed by dupword linter fix ([3983185](https://github.com/kausys/apikit/commit/3983185f0afe5d40b5ee3d676accc766263f2eb9))
