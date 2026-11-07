@domain:load-balancing @area:circuit-breaking @from:utils/circuit-breaker
Feature: Protecting the cache backend with a circuit breaker
  Operators rely on the proxy to stop calling a failing cache backend for a while, per domain, and to resume once it recovers, so a Redis outage does not stall every request.

  Background:
    Given Given a domain www.example.com is served by the proxy with a cache backend
    And And the domain's circuit_breaker settings are threshold 2, failurerate 0.5, interval 0s, timeout 60s and maxrequests 1 (the built-in defaults)

  @id:load-balancing.circuit-breaking.001 @rule:cache.backend-breaker.trips-on-failure-rate-after-threshold @verified @source:utils/circuit-breaker/circuit-breaker.go#cbReadyToTrip @test:cache/engine/client/client_test.go#TestCircuitBreakerWithPingTimeout
  Scenario Outline: The breaker opens only once enough calls were made and the failure rate is reached
    Given Given the breaker for www.example.com is closed
    And And the cache backend has received <calls> calls for the domain, of which <failures> failed
    When When the cache backend call that brings the totals to those numbers completes
    Then Then the breaker for www.example.com is <state>
    And And while it is open, further cache backend calls for the domain fail at once without reaching the backend

    Examples:
      | calls | failures | state  |
      | 1     | 1        | closed |
      | 2     | 1        | open   |
      | 2     | 2        | open   |
      | 4     | 1        | closed |

  @id:load-balancing.circuit-breaking.002 @rule:cache.backend-breaker.recovers-after-timeout @verified @source:utils/circuit-breaker/circuit-breaker.go#InitCircuitBreaker @test:cache/engine/client/client_test.go#TestCircuitBreakerWithPingTimeout
  Scenario: After the open period a trial call is let through and a success closes the breaker
    Given Given the breaker for www.example.com opened because the cache backend was failing
    And And the configured timeout has elapsed
    And And the cache backend is reachable again
    When When the proxy makes its next cache backend call for the domain
    Then Then that call reaches the cache backend and succeeds
    And And the breaker for www.example.com is closed again

  @id:load-balancing.circuit-breaking.003 @rule:cache.backend-breaker.state-changes-logged @inferred @source:utils/circuit-breaker/circuit-breaker.go#cbOnStateChange
  Scenario Outline: Every breaker state change is reported to the operator as a warning
    Given Given the breaker for www.example.com is closed
    When When the breaker for www.example.com changes from <from> to <to>
    Then Then the proxy logs a warning "Circuit Breaker '<domain id>' - Changed from <from> to <to>"

    Examples:
      | from      | to        |
      | closed    | open      |
      | open      | half-open |
      | half-open | closed    |
      | half-open | open      |

  @id:load-balancing.circuit-breaking.004 @rule:cache.backend-breaker.isolated-per-domain @inferred @source:utils/circuit-breaker/circuit-breaker.go#InitCircuitBreaker
  Scenario: A failing cache backend for one domain does not stop cache use for another
    Given Given www.example.com and shop.example.org are each served with their own upstream
    And And the breaker for www.example.com is open
    When When a client requests a page on shop.example.org
    Then Then the proxy still queries the cache backend for shop.example.org
    And And the breaker for shop.example.org stays closed
