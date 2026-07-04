# Changelog

## [0.8.0](https://github.com/kausys/apikit/compare/cmd-v0.7.2...cmd-v0.8.0) (2026-07-04)


### Features

* add authz code generation with groups support ([#9](https://github.com/kausys/apikit/issues/9)) ([07affb9](https://github.com/kausys/apikit/commit/07affb99caac26dc162bb61744ce434cd427f2f7))
* **authz:** accept repeatable --input to merge multiple CSVs ([#20](https://github.com/kausys/apikit/issues/20)) ([c0db8d0](https://github.com/kausys/apikit/commit/c0db8d066305df3d03a97e18c091a2a230a7557b))
* consolidate from 5 modules to 2 (root + cmd) ([#7](https://github.com/kausys/apikit/issues/7)) ([e81b38e](https://github.com/kausys/apikit/commit/e81b38e2de23f3472047e44355ac50f5a11f222f))
* initial release v0.2.0 ([27d280d](https://github.com/kausys/apikit/commit/27d280d94f8429fea31c889971680f274aa6a083))
* pluggable error renderer + ctx-aware response handling ([#16](https://github.com/kausys/apikit/issues/16)) ([a869232](https://github.com/kausys/apikit/commit/a869232b47cd1f2f4c85f22c0120b1b36ce139cb))


### Bug Fixes

* **cmd:** handler gen accepts directories and ./... patterns ([#18](https://github.com/kausys/apikit/issues/18)) ([40c1c94](https://github.com/kausys/apikit/commit/40c1c949d74f55aa984b8b9a498a403c9856505c))
* CodeQL integer conversion in handler codegen + swagger open-redirect ([#13](https://github.com/kausys/apikit/issues/13)) ([408dc9a](https://github.com/kausys/apikit/commit/408dc9ab0acbfe33d4143ff36056e1b8f336956f))
* move main package to cmd/apikit/ for correct binary name ([c022b19](https://github.com/kausys/apikit/commit/c022b19fd293820dda9061d9caef60a6d218e3c0))
* **scanner:** set apiKey securityScheme name from the name: property ([#24](https://github.com/kausys/apikit/issues/24)) ([4500836](https://github.com/kausys/apikit/commit/4500836e1049fa896abad1d8cc844f967d635c0f))
* support fully-qualified import paths for custom types ([#4](https://github.com/kausys/apikit/issues/4)) ([fbdfe38](https://github.com/kausys/apikit/commit/fbdfe38b2c392b166d7fc0a6876e1b9e8c56c863))
* **swagger:** remove -v shorthand collision with root --verbose flag ([#11](https://github.com/kausys/apikit/issues/11)) ([fa92b46](https://github.com/kausys/apikit/commit/fa92b466142d0a85d4ace13c7713db0613d30078))
