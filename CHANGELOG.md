# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]
- Added PartnerID and WebhookValidation to Config. [#83](https://github.com/xmidt-org/ancla/pull/83)

## [v0.3.1]
- Export DisablePartnerIDs from HandlerConfig. [#82](https://github.com/xmidt-org/ancla/pull/82)

## [v0.3.0]
- Added configurability for partnerIDs check and continued converting webhooks to 
internalWebhooks. [#80](https://github.com/xmidt-org/ancla/pull/80)
- Changed webhooks to internalWebhooks to enable the storing of partnerIDs. [#79](https://github.com/xmidt-org/ancla/pull/79)

## [v0.2.4]
- Update webhookValidator builder to fix http issue. [#77](https://github.com/xmidt-org/ancla/pull/77)

## [v0.2.3]
- Added http check in webhookValidator. [#75](https://github.com/xmidt-org/ancla/pull/75)

## [v0.2.2]
- Added validators for deviceID, Until, Duration, Events to webhookValidator. [#67](https://github.com/xmidt-org/ancla/pull/67)
- Updated decoding errors to return 400 status codes. [#72](https://github.com/xmidt-org/ancla/pull/72)

## [v0.2.1]
- Added webhookValidator. Validates the webhook's Config.URL, Config.AlternativeURLs, and FailureURL. [#65](https://github.com/xmidt-org/ancla/pull/65)
- Fix security warning by removing old jwt lib as direct dependency. [#66](https://github.com/xmidt-org/ancla/pull/66)

## [v0.2.0]
- Bumped argus, webpa-common versions. Updated metrics to be compatible. [#63](https://github.com/xmidt-org/ancla/pull/63)

## [v0.1.6]
- Support acquiring JWT token from Themis. [#59](https://github.com/xmidt-org/ancla/pull/59)

## [v0.1.5]
- Update Argus version with request context bugfix. [#55](https://github.com/xmidt-org/ancla/pull/55)

## [v0.1.4]
- Update to version of Argus compatible with optional OpenTelemetry tracing feature. [#51](https://github.com/xmidt-org/ancla/pull/51)

## [v0.1.3]
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

[Unreleased]: https://github.com/xmidt-org/ancla/compare/v0.3.1...HEAD
[v0.3.1]: https://github.com/xmidt-org/ancla/compare/0.3.0...v0.3.1
[v0.3.0]: https://github.com/xmidt-org/ancla/compare/0.2.4...v0.3.0
[v0.2.4]: https://github.com/xmidt-org/ancla/compare/0.2.3...v0.2.4
[v0.2.3]: https://github.com/xmidt-org/ancla/compare/0.2.2...v0.2.3
[v0.2.2]: https://github.com/xmidt-org/ancla/compare/0.2.1...v0.2.2
[v0.2.1]: https://github.com/xmidt-org/ancla/compare/0.2.0...v0.2.1
[v0.2.0]: https://github.com/xmidt-org/ancla/compare/0.1.6...v0.2.0
[v0.1.6]: https://github.com/xmidt-org/ancla/compare/0.1.5...v0.1.6
[v0.1.5]: https://github.com/xmidt-org/ancla/compare/0.1.4...v0.1.5
[v0.1.4]: https://github.com/xmidt-org/ancla/compare/0.1.3...v0.1.4
[v0.1.3]: https://github.com/xmidt-org/ancla/compare/0.1.2...v0.1.3
[v0.1.2]: https://github.com/xmidt-org/ancla/compare/0.1.1...v0.1.2
[v0.1.1]: https://github.com/xmidt-org/ancla/compare/0.1.0...v0.1.1
[v0.1.0]: https://github.com/xmidt-org/ancla/compare/0.0.0...v0.1.0
