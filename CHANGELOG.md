# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.4.0] - 2026-08-23

### Changed
- CI: migrated workflows to shared workflows from `upcloud-tools/gha-actions` with git submodule for composite actions.
- CI: container build uses a `buildah` script (`build-image.sh`) instead of a multi-stage `Containerfile`.

## [1.3.1] - 2025-07-28

### Changed

- Bump `github.com/cert-manager/cert-manager` from v1.19.5 to v1.21.0.
- Upgraded indirect dependencies to match cert-manager v1.21.0 requirements and fix some security issues.

## [1.3.0] - 2025-07-28

### Added

- Unit tests using `httptest` for the Bunny API client (resolve zone, add/delete records).
- HTTP context propagation through all DNS API client methods.
- Curated golangci-lint linters (bodyclose, nilerr, dupword, funlen, gocyclo).
- CHANGELOG.md

### Changed

- Internal refactoring: Bunny API client moved to `internal/bunny` subpackage.
- Go naming convention fixes (acronym casing, identifier renames).
- API error response body truncated to 200 characters in error messages.

## [1.2.0] - 2025-07-24

### Changed

- Add support for multiple zones and remove static zoneId by @aardbol in #9

## [1.0.6] - 2025-07-13

### Fixed

- Minor bug fixes and dependency updates.

## [1.0.5] - 2025-07-09

### Added

- Initial release of cert-manager-webhook-bunny.

