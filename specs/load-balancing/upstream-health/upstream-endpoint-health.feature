@domain:load-balancing @area:upstream-health @from:server/balancer
Feature: Keeping unhealthy endpoints out of rotation
  Lets an operator rely on periodic probes so traffic only reaches upstream endpoints that answer as expected.

  Background:
    Given the operator has configured a domain whose upstream lists the endpoints 10.0.0.1 and 10.0.0.2

  @id:load-balancing.upstream-health.001 @rule:load-balancing.health.endpoint-in-rotation-only-on-accepted-status @verified @source:server/balancer/balancer.go#DoHealthCheck @test:server/balancer/balancer_func_test.go#TestHealthCheckWithCustomPort
  Scenario Outline: An endpoint stays in rotation only while its probe answers with an accepted status
    Given the accepted health-check status codes are <accepted>
    And 10.0.0.2 is in rotation
    When the periodic health check probes 10.0.0.2 and it <answers>
    Then 10.0.0.2 is <state> for new requests

    Examples:
      | accepted      | answers                                                   | state                 |
      | 200 (default) | responds 200                                              | kept in rotation      |
      | 200 (default) | responds 503                                              | taken out of rotation |
      | 200 (default) | responds 301 redirect                                     | taken out of rotation |
      | 200, 301      | responds 301 redirect                                     | kept in rotation      |
      | 200 (default) | does not answer within the timeout (5 seconds by default) | taken out of rotation |

  @id:load-balancing.upstream-health.002 @rule:load-balancing.health.recovered-endpoint-returns @inferred @source:server/balancer/balancer.go#CheckHealth
  Scenario Outline: An endpoint that passes its probe again goes back into rotation
    Given 10.0.0.2 was taken out of rotation after a failed probe
    And the health-check interval is <interval>
    When 10.0.0.2 answers the next probe with an accepted status
    Then 10.0.0.2 receives new requests again within one interval

    Examples:
      | interval             |
      | not set (30 seconds) |
      | 10s                  |

  @id:load-balancing.upstream-health.003 @rule:load-balancing.health.endpoints-trusted-until-first-probe @inferred @source:server/balancer/balancer.go#convertEndpoints
  Scenario: Endpoints receive traffic from start-up until their first probe
    Given 10.0.0.2 is not reachable
    And the proxy has just started
    When a client sends requests to the domain before the first health-check interval elapses
    Then some requests are sent to 10.0.0.2

  @id:load-balancing.upstream-health.004 @rule:load-balancing.health.probe-uses-upstream-scheme-and-port @verified @source:server/balancer/balancer.go#healthCheckFor @test:server/balancer/healthcheck_scheme_test.go#TestHealthCheckInheritsTheUpstreamScheme @test:server/balancer/balancer_func_test.go#TestHealthCheckWithCustomPort @questionable
  Scenario Outline: The health probe uses the upstream's scheme and port unless the health check sets its own
    Given the upstream is declared with scheme "http" and port 8080
    And the health check sets <hc_setting>
    When the periodic health check probes 10.0.0.1
    Then the probe is sent to <probe_target>

    Examples:
      | hc_setting                  | probe_target                                                                     |
      | no scheme and no port       | https://10.0.0.1:8080 (the built-in https default wins over the upstream's http) |
      | scheme "http" and port 8000 | http://10.0.0.1:8000                                                             |

  @id:load-balancing.upstream-health.005 @rule:load-balancing.health.probe-verifies-tls-unless-allowed-insecure @inferred @source:server/balancer/balancer.go#getClient
  Scenario Outline: An HTTPS endpoint with an untrusted certificate fails its probe unless insecure probing is allowed
    Given 10.0.0.1 is probed over https and presents a self-signed certificate
    And the health check's allow_insecure is <allow_insecure>
    When the periodic health check probes 10.0.0.1
    Then 10.0.0.1 is <state>

    Examples:
      | allow_insecure  | state                                                    |
      | false (default) | taken out of rotation                                    |
      | true            | kept in rotation when it answers with an accepted status |
