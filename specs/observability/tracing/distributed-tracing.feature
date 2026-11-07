@domain:observability @area:tracing @from:telemetry/tracing
Feature: Tracing proxied requests
  Lets operators follow each request through the proxy and on into upstream services in their Jaeger tracing backend, so they can diagnose latency and failures end to end.

  Background:
    Given the operator has turned tracing on with a Jaeger collector endpoint of "http://jaeger.example.com:14268/api/traces"

  @id:observability.tracing.001 @rule:observability.tracing.traces-identify-service-version-env @inferred @source:telemetry/tracing/provider.go#NewJaegerProvider
  Scenario: Exported traces identify the proxy, its version and its environment
    Given the proxy is version "1.2.3"
    And the TRACING_ENV environment variable is "staging"
    When a client requests "https://www.example.com/products" through the proxy
    Then the Jaeger collector receives a trace for the request
    And the trace's service name is "go-proxy-cache"
    And the trace's service version attribute is "1.2.3"
    And the trace's service environment attribute is "staging"

  @id:observability.tracing.002 @rule:observability.tracing.sampling-ratio @inferred @source:telemetry/tracing/provider.go#NewJaegerProvider @questionable
  Scenario Outline: Only the configured share of requests is traced
    Given the operator has set the tracing sampling ratio to <ratio>
    When clients send 1000 requests through the proxy
    Then about <traced> of the requests appear as traces in Jaeger

    Examples:
      | ratio | traced |
      | 1.0   | 1000   |
      | 0.25  | 250    |
      | 0     | 0      |

  @id:observability.tracing.003 @rule:observability.tracing.context-propagated-to-upstream @inferred @source:telemetry/tracing/span.go#Inject
  Scenario: Upstream services receive the trace context so their spans join the same trace
    Given a client sends a request carrying a W3C "traceparent" header with trace id "4bf92f3577b34da6a3ce929d0e0e4736"
    When the proxy forwards the request to the upstream
    Then the proxy's spans for the request belong to trace "4bf92f3577b34da6a3ce929d0e0e4736"
    And the upstream request carries a "traceparent" header with trace id "4bf92f3577b34da6a3ce929d0e0e4736"
    And the upstream request carries any W3C "baggage" the client sent

  @id:observability.tracing.004 @rule:observability.tracing.disabled-sends-nothing @inferred @source:server/server.go#Run
  Scenario: With tracing off, nothing is exported and no trace headers reach the upstream
    Given the operator has turned tracing off
    When a client requests "https://www.example.com/products" through the proxy
    Then the Jaeger collector receives no trace
    And the upstream request carries no "traceparent" header added by the proxy
