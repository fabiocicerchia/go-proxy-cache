@domain:load-balancing @area:balancing-strategies @from:server/balancer
Feature: Choosing an upstream endpoint
  Lets an operator spread a domain's traffic over several upstream endpoints with a chosen balancing algorithm, so no single backend carries all the load.

  Background:
    Given the operator has configured a domain whose upstream host is origin.example.com
    And the upstream lists the endpoints 10.0.0.1, 10.0.0.2 and 10.0.0.3, all passing their health checks

  @id:load-balancing.balancing-strategies.001 @rule:load-balancing.strategies.unknown-algorithm-uses-round-robin @inferred @source:server/balancer/balancer.go#Init @questionable
  Scenario Outline: A missing or unrecognised balancing algorithm behaves as round-robin
    Given the domain's balancing algorithm is <algorithm>
    When the proxy starts
    Then the domain's traffic is spread over the endpoints in round-robin order
    And no warning about the algorithm is reported to the operator

    Examples:
      | algorithm     |
      | not set       |
      | "weighted"    |
      | "Round-Robin" |

  @id:load-balancing.balancing-strategies.002 @rule:load-balancing.strategies.no-endpoint-falls-back-to-upstream-host @verified @source:server/balancer/balancer.go#GetUpstreamNode @test:server/balancer/balancer_test.go#TestGetUpstreamNodeUndefined @questionable
  Scenario Outline: With no endpoints available, requests go straight to the upstream host
    Given <situation>
    When a client sends a request to the domain
    Then the request is forwarded to origin.example.com
    And the client is not told that no endpoint was available

    Examples:
      | situation                                         |
      | the upstream lists no endpoints                   |
      | every listed endpoint has failed its health check |
