# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.4.2] - 2026-11-25

### Changed

- Bump Go toolchain from v1.26.8 to v1.27.4.
- Bump `github.com/cert-manager/cert-manager` from v1.21.2 to v1.21.3.
- Bump `go.opentelemetry.io/otel/exporters/otlp/otlptrace` from v1.40.0 to v1.47.0.
- Bump `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc` from v1.40.0 to v1.47.0.
- Bump indirect dependencies to implement security updates.
- CI: bump `upcloud-tools/gha-actions` shared workflows and composite-actions submodule to `f2881d6` (2026-08-27).
- CI: bump golangci-lint to v2.15.0 for Go 1.27 compatibility.

### Security

- Fix Go standard library vulnerabilities (including CVE-2026-111943, CVE-2026-97031, CVE-2026-78667 and CVE-2026-78669) by building with Go 1.27.4.
- Fix `golang.org/x/net/html` denial of service (CVE-2026-78661) and cross-site scripting (CVE-2026-97045) by bumping `golang.org/x/net` to v0.63.0.
- Fix OpenTelemetry exporter configuration logging information disclosure (GHSA-8wmf-6v46-5gfg) by bumping the OTLP trace exporters to v1.47.0.

## [1.4.1] - 2026-09-05

### Added

- Container images are now multi-arch (`linux/amd64` and `linux/arm64`).

### Changed

- Bump Go toolchain from v1.26.5 to v1.26.8.
- Bump `github.com/cert-manager/cert-manager` from v1.21.0 to v1.21.1.
- Bump `google.golang.org/grpc` from v1.82.1 to v1.83.1.
- Bump indirect dependencies to implement security updates.

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
