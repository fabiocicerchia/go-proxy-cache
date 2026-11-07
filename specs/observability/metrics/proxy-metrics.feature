@domain:observability @area:metrics @from:telemetry/metrics
Feature: Monitoring the proxy through its metrics
  Gives operators Prometheus series on traffic, cache effectiveness and upstream health, without exposing what visitors requested.

  Background:
    Given the proxy publishes its metrics on the internal /metrics endpoint

  @id:observability.metrics.001 @rule:observability.metrics.request-labels-bounded-and-private @verified @source:telemetry/metrics/enterprise.go#IncWholeRequest @test:telemetry/metrics/enterprise_unit_test.go#TestWholeRequestResponseSeriesAreBounded
  Scenario Outline: Request metrics never reveal the requested URL and stay bounded
    Given the proxy serves the domain "example.com"
    When a visitor sends a <method> request for "/page/7?token=secret7" to host "example.com:8443"
    Then the gpcee_http_request series is labelled host "example.com" and method "<shown_method>"
    And no label on /metrics contains "/page/7" or "secret7"
    And repeated requests to different paths add to the same series instead of creating new ones

    Examples:
      | method | shown_method |
      | GET    | GET          |
      | PURGE  | PURGE        |
      | FOO7   | OTHER        |

  @id:observability.metrics.002 @rule:observability.metrics.responses-counted-by-status-class @inferred @source:telemetry/metrics/metrics.go#IncStatusCode
  Scenario Outline: Every response is counted by its exact status and its status class
    When the proxy answers a request with status <status>
    Then gpc_status_codes_total for code "<status>" <change>
    And gpc_request_<class>_total <change>
    And gpc_request_sum_total <change>

    Examples:
      | status                     | class | change          |
      | 101 (WebSocket upgrade)    | 1xx   | does not change |
      | 204                        | 2xx   | goes up by 1    |
      | 301                        | 3xx   | goes up by 1    |
      | 404                        | 4xx   | goes up by 1    |
      | 503                        | 5xx   | goes up by 1    |

  @id:observability.metrics.003 @rule:observability.metrics.upstream-health-status @inferred @source:telemetry/metrics/enterprise.go#SetUpstreamServerHealthChecksHealthy @questionable
  Scenario Outline: Each upstream's latest health check result is published
    Given the upstream "backend-1.example.com:8080" of "example.com" is health-checked
    When a health check finds the upstream <outcome>
    Then gpcee_upstream_server_health_checks_status for that upstream is <status>
    And <counter> for that upstream goes up by 1

    Examples:
      | outcome                                          | status | counter                                       |
      | answering with an accepted status                | 1      | gpcee_upstream_server_health_checks_checks    |
      | answering with a status not in the accepted list | 0      | gpcee_upstream_server_health_checks_unhealthy |
      | unreachable                                      | -1     | gpcee_upstream_server_health_checks_fails     |

  @id:observability.metrics.004 @rule:observability.metrics.build-info @inferred @source:telemetry/metrics/enterprise.go#SetBuildInfo
  Scenario: The running version and commit are published
    Given the proxy was built as version "1.4.0" from commit "a1b2c3d"
    When the operator starts the proxy
    Then gpcee_generic_build_info with version "1.4.0" and git_commit "a1b2c3d" reads 1

  @id:observability.metrics.005 @rule:observability.metrics.up-reflects-cache-backend-health @inferred @source:telemetry/metrics/enterprise.go#SetUp @questionable
  Scenario Outline: The up metric reflects the cache backend health seen by the last health check
    Given the cache backend for every domain is <backend>
    When a monitor calls the /healthcheck endpoint
    Then gpcee_generic_up reads <up>

    Examples:
      | backend                             | up |
      | reachable                           | 1  |
      | unreachable for at least one domain | 0  |

  @id:observability.metrics.006 @rule:observability.metrics.instance-and-env-labels @inferred @source:telemetry/metrics/metrics.go#baseLabels
  Scenario: Every metric identifies the reporting machine and environment
    Given the proxy runs on machine "edge-01" with TRACING_ENV set to "production"
    When the operator reads any gpc_ or gpcee_ series on /metrics
    Then the series carries label hostname "edge-01"
    And the series carries label env "production"
