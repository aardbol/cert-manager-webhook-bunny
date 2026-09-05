# Chart Changelog

All notable changes to the Helm chart will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.4.2] - 2026-09-05

### Changed
- Bump app version to `v1.4.1`.

## [1.4.1] - 2026-08-24

### Added
- Chart-specific CHANGELOG.md

### Changed
- Improved README.md for artifacthub

## [1.4.0] - 2026-08-23

### Added
- `values.schema.json` for input validation.

### Changed
- `appVersion` now includes `v` prefix (e.g., `v1.4.0`).

### Removed
- `image.hash` field removed, use `image.tag` only.
