# Changelog

## [0.8.1](https://github.com/kausys/apikit/compare/v0.8.0...v0.8.1) (2026-07-19)


### Bug Fixes

* **http:** add GetCookie for cookie extractors ([#37](https://github.com/kausys/apikit/issues/37)) ([df9bf39](https://github.com/kausys/apikit/commit/df9bf39b5b55b54979bd132d2ab38f3b1d2cc5bb))

## [0.8.0](https://github.com/kausys/apikit/compare/v0.7.3...v0.8.0) (2026-07-06)


### Features

* **types:** register uuid.UUID header/query/path extractor ([#34](https://github.com/kausys/apikit/issues/34)) ([eea194a](https://github.com/kausys/apikit/commit/eea194a2912940d228252051e153736845100574))


### Bug Fixes

* **swagger:** preserve caller-defined spec order in dropdown ([#35](https://github.com/kausys/apikit/issues/35)) ([0256c9f](https://github.com/kausys/apikit/commit/0256c9fafc00c2389a1d149a7ba029ca980798dd))

## [0.7.3](https://github.com/kausys/apikit/compare/v0.7.2...v0.7.3) (2026-07-04)


### Bug Fixes

* **release:** use tag-separator so cmd anchors to cmd/v* (lockstep 0.7.3) ([#30](https://github.com/kausys/apikit/issues/30)) ([267683b](https://github.com/kausys/apikit/commit/267683b613047abde7460cb36426eb7f7a20861e))


### Miscellaneous

* **release:** keep root and cmd in lockstep (linked-versions) ([#28](https://github.com/kausys/apikit/issues/28)) ([dad3149](https://github.com/kausys/apikit/commit/dad3149178248aa740cdb915925cb2fcd25308a5))
* **release:** lockstep cmd with root by mirroring the release tag ([#32](https://github.com/kausys/apikit/issues/32)) ([df6c02a](https://github.com/kausys/apikit/commit/df6c02a0c922b740294114cbaa56345b1e242aa1))
* **release:** manage cmd submodule as its own release-please component ([#26](https://github.com/kausys/apikit/issues/26)) ([71d31c5](https://github.com/kausys/apikit/commit/71d31c570d2f0032215e2374530ad4fc464cef20))

## [0.7.2](https://github.com/kausys/apikit/compare/v0.7.1...v0.7.2) (2026-07-04)


### Bug Fixes

* **scanner:** set apiKey securityScheme name from the name: property ([#24](https://github.com/kausys/apikit/issues/24)) ([4500836](https://github.com/kausys/apikit/commit/4500836e1049fa896abad1d8cc844f967d635c0f))

## [0.7.1](https://github.com/kausys/apikit/compare/v0.7.0...v0.7.1) (2026-07-04)


### Bug Fixes

* **http:** emit multiple Set-Cookie headers via HttpResponse.Cookies ([#22](https://github.com/kausys/apikit/issues/22)) ([f95f562](https://github.com/kausys/apikit/commit/f95f56272957c00e0d80a8a8a9c2c6b17e490153))

## [0.7.0](https://github.com/kausys/apikit/compare/v0.6.1...v0.7.0) (2026-07-03)


### Features

* **authz:** accept repeatable --input to merge multiple CSVs ([#20](https://github.com/kausys/apikit/issues/20)) ([c0db8d0](https://github.com/kausys/apikit/commit/c0db8d066305df3d03a97e18c091a2a230a7557b))

## [0.6.1](https://github.com/kausys/apikit/compare/v0.6.0...v0.6.1) (2026-07-02)


### Bug Fixes

* **cmd:** handler gen accepts directories and ./... patterns ([#18](https://github.com/kausys/apikit/issues/18)) ([40c1c94](https://github.com/kausys/apikit/commit/40c1c949d74f55aa984b8b9a498a403c9856505c))

## [0.6.0](https://github.com/kausys/apikit/compare/v0.5.2...v0.6.0) (2026-06-29)


### Features

* pluggable error renderer + ctx-aware response handling ([#16](https://github.com/kausys/apikit/issues/16)) ([a869232](https://github.com/kausys/apikit/commit/a869232b47cd1f2f4c85f22c0120b1b36ce139cb))

## [0.5.2](https://github.com/kausys/apikit/compare/v0.5.1...v0.5.2) (2026-05-07)


### Bug Fixes

* CodeQL integer conversion in handler codegen + swagger open-redirect ([#13](https://github.com/kausys/apikit/issues/13)) ([408dc9a](https://github.com/kausys/apikit/commit/408dc9ab0acbfe33d4143ff36056e1b8f336956f))

## [0.5.1](https://github.com/kausys/apikit/compare/v0.5.0...v0.5.1) (2026-04-16)


### Bug Fixes

* **swagger:** remove -v shorthand collision with root --verbose flag ([#11](https://github.com/kausys/apikit/issues/11)) ([fa92b46](https://github.com/kausys/apikit/commit/fa92b466142d0a85d4ace13c7713db0613d30078))

## [0.5.0](https://github.com/kausys/apikit/compare/v0.4.0...v0.5.0) (2026-03-15)


### Features

* add authz code generation with groups support ([#9](https://github.com/kausys/apikit/issues/9)) ([07affb9](https://github.com/kausys/apikit/commit/07affb99caac26dc162bb61744ce434cd427f2f7))

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
