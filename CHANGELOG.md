# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.14.1] - 2022-04-07

### Changed

- Upgrade OTel to `v1.6.2`. (#82)

## [0.14.0] - 2022-04-05

### ⚠️ Notice ⚠️

This update is a breaking change of `Open`, `OpenDB`, `Register`, `WrapDriver` and `RegisterDBStatsMetrics` methods.
Code instrumented with these methods will need to be modified.

### Removed

- Remove `dbSystem` parameter from all exported functions. (#80)

## [0.13.0] - 2022-04-04

### Added

- Add Metrics support. (#74)
- Add `Open` and `OpenDB` methods to instrument `database/sql`. (#77)

### Changed

- Upgrade OTel to `v1.6.0/v0.28.0`. (#74)
- Upgrade OTel to `v1.6.1`. (#76)

## [0.12.0] - 2022-03-18

### Added

- Covering connector's connect method with span. (#66)
- Add Go 1.18 to supported versions. (#69)

### Changed

- Upgrade OTel to `v1.5.0`. (#67)

## [0.11.0] - 2022-02-22

### Changed

- Upgrade OTel to `v1.4.1`. (#61)

## [0.10.0] - 2021-12-13

### Changed

- Upgrade OTel to `v1.2.0`. (#50)
- Upgrade OTel to `v1.3.0`. (#54)

## [0.9.0] - 2021-11-05

### Changed

- Upgrade OTel to v1.1.0. (#37)

## [0.8.0] - 2021-10-13

### Changed

- Upgrade OTel to v1.0.1. (#33)

## [0.7.0] - 2021-09-21

### Changed

- Upgrade OTel to v1.0.0. (#31)

## [0.6.0] - 2021-09-06

### Added

- Added RecordError to SpanOption. (#23)
- Added DisableQuery to SpanOption. (#26)

### Changed

- Upgrade OTel to v1.0.0-RC3. (#29)

## [0.5.0] - 2021-08-02

### Changed

- Upgrade OTel to v1.0.0-RC2. (#18)

## [0.4.0] - 2021-06-25

### Changed

- Upgrade to v1.0.0-RC1 of `go.opentelemetry.io/otel`. (#15)

## [0.3.0] - 2021-05-13

### Added

- Add AllowRoot option to prevent backward incompatible. (#13)

### Changed

- Upgrade to v0.20.0 of `go.opentelemetry.io/otel`. (#8)
- otelsql will not create root spans in absence of existing spans by default. (#13)

## [0.2.1] - 2021-03-28

### Fixed

- otelsql does not set the status of span to Error while recording error. (#5)

## [0.2.0] - 2021-03-24

### Changed

- Upgrade to v0.19.0 of `go.opentelemetry.io/otel`. (#3)

## [0.1.0] - 2021-03-23

This is the first release of otelsql.
It contains instrumentation for trace and depends on OTel `v0.18.0`.

### Added

- Instrumentation for trace.
- CI files.
- Example code for a basic usage.
- Apache-2.0 license.

[Unreleased]: https://github.com/XSAM/otelsql/compare/v0.14.1...HEAD
[0.14.1]: https://github.com/XSAM/otelsql/releases/tag/v0.14.1
[0.14.0]: https://github.com/XSAM/otelsql/releases/tag/v0.14.0
[0.13.0]: https://github.com/XSAM/otelsql/releases/tag/v0.13.0
[0.12.0]: https://github.com/XSAM/otelsql/releases/tag/v0.12.0
[0.11.0]: https://github.com/XSAM/otelsql/releases/tag/v0.11.0
[0.10.0]: https://github.com/XSAM/otelsql/releases/tag/v0.10.0
[0.9.0]: https://github.com/XSAM/otelsql/releases/tag/v0.9.0
[0.8.0]: https://github.com/XSAM/otelsql/releases/tag/v0.8.0
[0.7.0]: https://github.com/XSAM/otelsql/releases/tag/v0.7.0
[0.6.0]: https://github.com/XSAM/otelsql/releases/tag/v0.6.0
[0.5.0]: https://github.com/XSAM/otelsql/releases/tag/v0.5.0
[0.4.0]: https://github.com/XSAM/otelsql/releases/tag/v0.4.0
[0.3.0]: https://github.com/XSAM/otelsql/releases/tag/v0.3.0
[0.2.1]: https://github.com/XSAM/otelsql/releases/tag/v0.2.1
[0.2.0]: https://github.com/XSAM/otelsql/releases/tag/v0.2.0
[0.1.0]: https://github.com/XSAM/otelsql/releases/tag/v0.1.0
