@domain:observability @area:health-endpoint @from:server/handler
Feature: Health check
  Lets operators and orchestrators see whether the proxy and every domain's cache store are healthy.

  @id:observability.health-endpoint.001 @rule:observability.health.reports-cache-store @verified @source:server/handler/healthcheck.go#HandleHealthcheck @test:server/handler/healthcheck_test.go#TestHealthcheckWithRedis @test:server/handler/healthcheck_test.go#TestHealthcheckWithoutRedis
  Scenario Outline: The health check fails when any domain's cache store is unreachable
    Given <store_state>
    When the orchestrator requests /healthcheck
    Then the response status is <status>
    And the body contains "HTTP OK"
    And the body contains "<redis_line>"

    Examples:
      | store_state                           | status | redis_line |
      | every domain's cache store answers    | 200    | REDIS OK   |
      | the cache store of one domain is down | 500    | REDIS KO   |
