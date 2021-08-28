# Changelog

## v1.1.0 (2021-08-28)

### New

- Added new label for HIT/MISS in the logs. [Fabio Cicerchia]

### Changes

- Linted. [Fabio Cicerchia]
- Upgraded package.json. [Fabio Cicerchia]
- Refactoring to use DomainID consistenly in various places + changed some logging. [Fabio Cicerchia]
- Displaying omitted when printing configuration settings. [Fabio Cicerchia]

### Fix

- Refactoring to avoid to share HTTP headers with sensitive information in redis key. [Fabio Cicerchia]
- Automatically push to Docker Hub, closed #103. [Fabio Cicerchia]
- Fix CodeClimate broken coverage, closed #44. [Fabio Cicerchia]
- Added missing docs for golint. [Fabio Cicerchia]
- Generating dummy tls certificates for tests. [Fabio Cicerchia]
- Replacing correctly the cache_status_label variable. [Fabio Cicerchia]
- Lowered priority from WARN to INFO in case of MISS in redis + fixed one test + fixed wrong wildcard schema being passed to the proxy. [Fabio Cicerchia]
- Using the correct redis connection in case of wildcard scheme matching. [Fabio Cicerchia]
- Using correct tls certificates on test/full-setup. [Fabio Cicerchia]

### Other

- Create codeql-analysis.yml. [Fabio Cicerchia]
- Dev: added comments to explain critical settings related to issue #35. [Fabio Cicerchia]

## v1.0.1 (2021-08-26)

### Other

- Fixed golang version base image for docker. [Fabio Cicerchia]
- Corrected changelog for v1.0.0. [Fabio Cicerchia]

## v1.0.0 (2021-08-26)

### New

- Added tests for CONNECT HTTP method [Fabio Cicerchia]
- RedirectToHTTPS can use customizable RedirectStatusCode per domain [Fabio Cicerchia]
- Customize TLS config per domain: CurvePreferences, MinVersion, MaxVersion, and CipherSuites [Fabio Cicerchia]
- Added more tests and increased coverage [Fabio Cicerchia]
- Added the required mention in the readme for OpenSSL. [Fabio Cicerchia]

### Changes

- Refactoring + tests. [Fabio Cicerchia]
- Skipping HTTPS server configuration in case no TLS certificates are provided [Fabio Cicerchia]
- Upgraded docker images version for redis, node, and nginx [Fabio Cicerchia]
- Refactored error handling [Fabio Cicerchia]
- Removed stutter namings [Fabio Cicerchia]
- Refactored redis mutex locks [Fabio Cicerchia]
- Refactored redis circuit breaker calls [Fabio Cicerchia]
- Refactored config to get values automatically from env vars [Fabio Cicerchia]
- Refactored utils functions [Fabio Cicerchia]
- **Breaking change**: CACHE_ALLOWED_STATUSES and CACHE_ALLOWED_METHODS will now support spaces instead of commas [Fabio Cicerchia]
- Refactored domainConfig to be included in RequestCall (reduced coupling with config.go) [Fabio Cicerchia]
- Replaced SIGKILL handling with SIGTERM [Fabio Cicerchia]
- Refactored transport.handleBody as there was no real error (removed also shouldPanicOnCopyError and ctx) [Fabio Cicerchia]

### Fix

- Fixed missing TLS files + Refactoring (#102) [Fabio Cicerchia]
- Fixed documentation [Fabio Cicerchia]
- Fixed missing depencencies for github action tests [Fabio Cicerchia]
- Fixed nginx configuration for ssl settings [Fabio Cicerchia]

### Other

- Linting code using several linters [Fabio Cicerchia]
- Resolved some TODOs [Fabio Cicerchia]
- Version bump for golang image. [Fabio Cicerchia]
- Build(deps): bump alpine from 3.14.0 to 3.14.1. [dependabot[bot]]
- Bump ws from 7.4.1 to 7.4.6 in /test/full-setup. [dependabot[bot]]
- Bump alpine from 3.13.5 to 3.14.0. [dependabot[bot]]
- Bump golang from 1.16.1-alpine3.12 to 1.16.3-alpine3.12. [dependabot[bot]]
- Bump alpine from 3.13.2 to 3.13.5. [dependabot[bot]]
- Bump alpine from 3.13.0 to 3.13.2. [dependabot[bot]]
- Bump golang from 1.15.6-alpine3.12 to 1.16.1-alpine3.12. [dependabot[bot]]
- Bump alpine from 3.12.3 to 3.13.0. [dependabot[bot]]

## v0.3.0 (2021-01-07)

### New

- Missing locks on write/read on redis, closes #45. [Fabio Cicerchia]
- Add tests for HTTP2 Push, closes #75. [Fabio Cicerchia]
- Added tests for websockets. [Fabio Cicerchia]
- Kubernetes Example, closes #60. [Fabio Cicerchia]
- Add readthedocs.org, closes #30. [Fabio Cicerchia]
- Use X-Forwarded-Proto, closes #26. [Fabio Cicerchia]

### Changes

- Add tests for etags http headers, closes #71. [Fabio Cicerchia]

### Fix

- Test logging to file, closes #73. [Fabio Cicerchia]

### Other

- Updated changelog for v0.3.0. [Fabio Cicerchia]
- Updated changelog + fixes on makefile. [Fabio Cicerchia]
- Added missing file for readthedocs.io. [Fabio Cicerchia]
- Fixed some todos. [Fabio Cicerchia]
- Refactoring tests. [Fabio Cicerchia]

## v0.2.0 (2020-12-18)

### New

- WebSockets, closes #16 (#63) [Fabio Cicerchia]
- Handle ETags, closes #10 (#62) [Fabio Cicerchia]

### Other

- Changelog. [Fabio Cicerchia]
- Bump alpine from 3.12.2 to 3.12.3 (#66) [dependabot[bot]]
- HTTP2 Pusher interface is broken (#65) [Fabio Cicerchia]

## v0.1.0 (2020-12-15)

### Other

- Updated changelog. [Fabio Cicerchia]
- Bump alpine from 3.12.1 to 3.12.2 (#61) [dependabot[bot]]
- Add license scan report and status (#64) [fossabot]
- Disabled codeclimate in pipeline, as it is not working and still waiting from customer support. [Fabio Cicerchia]

## v0.1.0-beta1 (2020-12-08)

### Changes

- Benchmarks, closed #14 (#59) [Fabio Cicerchia]
- Refactor to have an upstream pool, closed #47. [Fabio Cicerchia]

### Fix

- Fix CodeClimate broken coverage, #44. [Fabio Cicerchia]
- Tests for chunking, closed #43. [Fabio Cicerchia]
- Add tags in Configuration struct, closed #41. [Fabio Cicerchia]

### Other

- Fixed paambaati/codeclimate-action@v2.7.4. [Fabio Cicerchia]

## v0.1.0-alpha1 (2020-12-04)

### New

- Make release, closed #6. [Fabio Cicerchia]
- Docker docs, closed #34. [Fabio Cicerchia]
- Check configuration validity, closed #7. [Fabio Cicerchia]
- Make health-check optional, with config flag, closed #29. [Fabio Cicerchia]
- Set log line format in config, closed #32. [Fabio Cicerchia]

### Fix

- Fixing to achieve good score on BetterCodeHub, #40. [Fabio Cicerchia]
- Broken tests for logs. [Fabio Cicerchia]

### Other

- Bump golang from 1.15.5-alpine3.12 to 1.15.6-alpine3.12 (#39) [dependabot[bot]]
- Fixed broken insecurebridge. [Fabio Cicerchia]
- Disabled tests in dockerfile. [Fabio Cicerchia]
- Production-Ready (#5) [Fabio Cicerchia]
- Fix codecov in ci, fixing tests, added script flags, added tlsfuzzer, docs. [Fabio Cicerchia]
- Refactoring. [Fabio Cicerchia]
- Added changelog. [Fabio Cicerchia]
- Refactoring + CII Best Practices. [Fabio Cicerchia]
- Refactoring. [Fabio Cicerchia]
- Coverage. [Fabio Cicerchia]
- Refactoring. [Fabio Cicerchia]
- Multi-domains. [Fabio Cicerchia]
- Added graceful shutdown, small refactoring, gzip. [Fabio Cicerchia]
- Refactoring. [Fabio Cicerchia]
- Circuit breaker. [Fabio Cicerchia]
- Refactoring. [Fabio Cicerchia]
- Fixed image logo. [Fabio Cicerchia]
- Restored code. [Fabio Cicerchia]
- Wip chunks. [Fabio Cicerchia]
- Changed misleading description. [Fabio Cicerchia]
- Refactoring. [Fabio Cicerchia]
- Update FUNDING.yml. [Fabio Cicerchia]
- Refactoring. [Fabio Cicerchia]
- Added end-to-end tests, refactoring, minor fixes, improved logging, added docs, http2https. [Fabio Cicerchia]
- Small fixes. [Fabio Cicerchia]
- Refactoring + tests. [Fabio Cicerchia]
- Https, http2, readme, coverage. [Fabio Cicerchia]
- Added export methods comments. [Fabio Cicerchia]
- Refactoring + tests. [Fabio Cicerchia]
- Added healthcheck + purge method. [Fabio Cicerchia]
- Refactoring. [Fabio Cicerchia]
- Added support for expires header. [Fabio Cicerchia]
- Added yaml config. [Fabio Cicerchia]
- Added allowed statuses and methods. [Fabio Cicerchia]
- Added functional tests. [Fabio Cicerchia]
- Refactoring. [Fabio Cicerchia]
- Switch from gob to msgpack. [Fabio Cicerchia]
- Added tests. [Fabio Cicerchia]
- Bump golang from 1.15.4-alpine3.12 to 1.15.5-alpine3.12. [dependabot[bot]]
- Small refactoring. [Fabio Cicerchia]
- Wip for vary header. [Fabio Cicerchia]
- Refactoring, added tests, saving data into redis with gob, added load balancing (roundrobin), improved CI, added tests in dockerfile, linted, added check for race conditions, added config struct. [Fabio Cicerchia]
- Fixed dockerfile build. [Fabio Cicerchia]
- Linting. [Fabio Cicerchia]
- Changed readme badges. [Fabio Cicerchia]
- Refactoring tests. [Fabio Cicerchia]
- Adding tests. [Fabio Cicerchia]
- Added TTL based on s-maxage or max-age. [Fabio Cicerchia]
- Initial commit. [Fabio Cicerchia]
