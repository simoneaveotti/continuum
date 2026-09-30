# Changelog

## [0.9.1] - 2026-09-30

### Fixed

- Remote HTTPS authentication failures during fetch, pull, and push now explain
  that a token may be expired, revoked, or invalid and tell users to renew it,
  reauthenticate, and update saved Git credentials. Context-loading warnings
  use the same guidance.
- SSH public-key failures now suggest checking key loading and repository
  authorization. Network and push-policy errors are not labeled token failures.
- Authentication guidance avoids exposing raw Git diagnostics or credential-bearing
  URLs and retains the underlying error for internal inspection.

### Maintenance

- Split CLI argument parsers by command family and separated context loading,
  rendering, and collaboration helpers; removed unused wrappers.
- Expanded parser and context characterization tests, separated watch activity
  coverage from view tests, and moved path-containment tests to the filestore package.
- Documented credential renewal and the consequences of preferred sync strategies.

[0.9.1]: https://github.com/simoneaveotti/continuum/compare/v0.9.0...v0.9.1
