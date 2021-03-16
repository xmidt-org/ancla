# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]
- Add context arg to methods to allow tracing information to flow through. [#47](https://github.com/xmidt-org/ancla/pull/47) thanks to @Sachin4403

## [v0.1.2]
### Fixed
- A webhook item should expire based on the `Until` field. [#45](https://github.com/xmidt-org/ancla/pull/45)


## [v0.1.1]
### Added
- Add counter for special case for webhook decoding. [#34](https://github.com/xmidt-org/ancla/pull/34)
- Add tests to transport.go. [#36](https://github.com/xmidt-org/ancla/pull/36)
- Tests for watch and endpoint. [#37](https://github.com/xmidt-org/ancla/pull/37)
- Better Go documentation. [#39](https://github.com/xmidt-org/ancla/pull/39)

### Changed
- Bump argus client version to v0.3.11. [#20](https://github.com/xmidt-org/ancla/pull/20)
- Make it so AllWebhooks() doesn't filter on owner yet. [#31](https://github.com/xmidt-org/ancla/pull/31)
- Remove loggerGroup. [#31](https://github.com/xmidt-org/ancla/pull/31)
- Simplify owner logic for adding webhooks. [#36](https://github.com/xmidt-org/ancla/pull/36)
- Bump argus client version to v0.3.12. [#42](https://github.com/xmidt-org/ancla/pull/42)

### Fixed
- Fix linting warnings. [#6](https://github.com/xmidt-org/ancla/pull/6)
- Update package name in go files. [#25](https://github.com/xmidt-org/ancla/pull/25)
- Added Missing copyright and license file headers. [#33](https://github.com/xmidt-org/ancla/pull/33)

## [v0.1.0]
- Initial release

[Unreleased]: https://github.com/xmidt-org/ancla/compare/v0.1.2..HEAD
[v0.1.2]: https://github.com/xmidt-org/ancla/compare/0.1.1...v0.1.2
[v0.1.1]: https://github.com/xmidt-org/ancla/compare/0.1.0...v0.1.1
[v0.1.0]: https://github.com/xmidt-org/ancla/compare/0.0.0...v0.1.0
