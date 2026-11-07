# Specifications of go-proxy-cache

What this code does, one atomic Gherkin scenario per behaviour, read from the code by code-atlas.
Tags: `@verified` a test asserts it · `@inferred` read from the code only · `@gap` an entry point no test covers · `@human` written or amended by a person, never rewritten.

## Caching
Storing upstream responses and serving them again without hitting the upstream.

- **Cache backend**: [Running the cache on Redis](caching/cache-backend/cache-store-behaviour.feature) (3)
- **Cache lookup and storage**: [Serving pages from cache](caching/cache-lookup-store/serving-cached-pages.feature) (1), [Keeping private responses out of the shared cache](caching/cache-lookup-store/shared-cache-privacy.feature) (1)
- **Freshness and cache status**: [Choosing how long a response stays cached](caching/freshness/cache-lifetime.feature) (2), [Reporting cache status to clients](caching/freshness/cache-status-header.feature) (1), [Reporting and bypassing the cache](caching/freshness/cache-status.feature) (2), [Deciding how long a proxied response stays fresh](caching/freshness/response-freshness-lifetime.feature) (7)
- **Cache purging**: [Purging cached pages](caching/purging/cache-purge.feature) (3), [Purging a page or a whole host](caching/purging/purge-scope.feature) (3), [Purging cached pages](caching/purging/purging-cached-pages.feature) (1)

## Configuration
How operators describe the proxy's behaviour and per-site settings.

- **Config display**: [Showing the effective configuration at start-up](configuration/config-display/effective-config-display.feature) (3)
- **Config loading**: [Applying built-in defaults and rejecting invalid settings](configuration/config-loading/built-in-defaults-and-validation.feature) (3), [Loading and checking the configuration](configuration/config-loading/loading-configuration.feature) (5)
- **Config merging**: [Domain settings inheriting global settings](configuration/config-merging/domain-setting-inheritance.feature) (2), [Inheriting global settings per domain](configuration/config-merging/domain-settings-inheritance.feature) (6), [Per-domain settings](configuration/config-merging/per-domain-settings.feature) (3)
- **Multiple domains**: [Serving several domains from one proxy](configuration/multi-domain/domain-selection.feature) (3)

## Load Balancing
Spreading traffic across several upstream servers and avoiding unhealthy ones.

- **Balancing strategies**: [Choosing an upstream endpoint](load-balancing/balancing-strategies/choosing-an-upstream-endpoint.feature) (2)
- **Circuit breaking**: [Protecting the cache backend with a circuit breaker](load-balancing/circuit-breaking/cache-backend-circuit-breaking.feature) (4)
- **Upstream health**: [Keeping unhealthy endpoints out of rotation](load-balancing/upstream-health/upstream-endpoint-health.feature) (5)

## Observability
Seeing what the proxy is doing through logs, metrics, traces and health signals.

- **Health check endpoint**: [Health check](observability/health-endpoint/health-check.feature) (1)
- **Logging**: [Choosing log destinations and verbosity](observability/logging/log-outputs.feature) (5), [Request access logging](observability/logging/request-access-log.feature) (4), [Tamper-proof request logs](observability/logging/tamper-proof-request-logs.feature) (1)
- **Metrics**: [Monitoring the proxy through its metrics](observability/metrics/proxy-metrics.feature) (6)
- **Distributed tracing**: [Tracing proxied requests](observability/tracing/distributed-tracing.feature) (4)

## Operations and Developer Tooling
Starting, building, packaging and testing the proxy, plus shared helpers.

- **Build and release**: [Publishing prebuilt release binaries](operations/build-release/release-binaries.feature) (2)
- **Container image**: [Running the proxy from its container image](operations/container-image/container-image.feature) (3)
- **Server lifecycle**: [Starting the proxy from the command line](operations/server-lifecycle/command-line-start-up.feature) (6), [Stopping and reloading the proxy without dropping requests](operations/server-lifecycle/service-lifecycle.feature) (3), [Restarting without refusing connections](operations/server-lifecycle/zero-downtime-restarts.feature) (5)

## Reverse Proxying
Forwarding client requests to upstream servers and returning their responses.

- **Redirects and scheme handling**: [Redirecting plain HTTP to HTTPS](proxying/redirects/https-redirect.feature) (1), [Choosing the upstream scheme](proxying/redirects/upstream-scheme-setting.feature) (2)
- **Request forwarding**: [Accepting and forwarding requests for configured domains](proxying/request-forwarding/request-admission.feature) (4), [Limiting how long a request may take](proxying/request-forwarding/request-time-limits.feature) (3)
- **Response delivery**: [Compressed delivery](proxying/response-delivery/gzip-delivery.feature) (1), [Replaying stored responses to clients](proxying/response-delivery/replaying-stored-responses.feature) (4), [Delivering proxied responses](proxying/response-delivery/response-delivery.feature) (1)

## Security
Protecting traffic in transit and controlling who may use protected features.

- **Token authentication**: [Authenticating requests with JWT bearer tokens](security/jwt-auth/jwt-token-authentication.feature) (7)
- **Network listeners**: [Opening public and internal ports](security/listeners/listeners-and-internal-endpoints.feature) (3)
- **TLS and certificates**: [Serving HTTPS for hosted domains](security/tls-certificates/serving-https.feature) (7), [Serving HTTPS with modern TLS only](security/tls-certificates/tls-protocol-floor.feature) (1)
