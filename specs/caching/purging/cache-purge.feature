@domain:caching @area:purging @from:server/handler
Feature: Purging cached pages
  Lets authorised operators remove a stale page from the cache, and refuses everyone else.

  Background:
    Given the domain "shop.example.com" is configured for purging

  @id:caching.purging.002 @rule:caching.purge.requires-authorisation @verified @source:server/handler/purge.go#HandlePurge @test:server/handler/purge_auth_unit_test.go#TestIsPurgeAuthorizedDeniesMissingControls @test:server/handler/purge_auth_unit_test.go#TestIsPurgeAuthorizedSecret @test:server/handler/purge_auth_unit_test.go#TestIsPurgeAuthorizedSecretCustomHeader @test:server/handler/purge_auth_unit_test.go#TestIsPurgeAuthorizedIPAllowlist @test:server/handler/purge_auth_unit_test.go#TestIsPurgeAuthorizedBothMustPass
  Scenario Outline: Purges from callers who fail the configured access controls are refused
    Given the domain's purge access controls are <controls>
    When a caller from <caller_ip> sends PURGE /cart with <credentials>
    Then the response status is 403
    And the response body is "KO"
    And the cached page is still served

    Examples:
      | controls                                     | caller_ip    | credentials                            |
      | none configured                              | 203.0.113.7  | the correct X-Go-Proxy-Cache-Purge-Key |
      | secret "s3cr3t"                              | 203.0.113.7  | no purge key header                    |
      | secret "s3cr3t"                              | 203.0.113.7  | X-Go-Proxy-Cache-Purge-Key "nope"      |
      | secret "s3cr3t" in header X-Custom-Purge     | 203.0.113.7  | X-Go-Proxy-Cache-Purge-Key "s3cr3t"    |
      | allowlist 203.0.113.0/24                     | 198.51.100.9 | X-Forwarded-For "203.0.113.7"          |
      | secret "s3cr3t" and allowlist 203.0.113.0/24 | 198.51.100.9 | X-Go-Proxy-Cache-Purge-Key "s3cr3t"    |

  @id:caching.purging.003 @rule:caching.purge.removes-cached-page @verified @source:server/handler/purge.go#HandlePurge @test:server/handler/purge_test.go#TestEndToEndCallPurge
  Scenario: An authorised purge makes the next visit fetch the page fresh
    Given the page /cart is cached
    When an authorised operator sends PURGE /cart
    Then the response status is 200
    And the response body is "OK"
    And the next GET /cart carries X-Go-Proxy-Cache-Status "MISS"

  @id:caching.purging.004 @rule:caching.purge.distinguishes-nothing-cached-from-failure @verified @source:server/handler/purge.go#HandlePurge @test:server/handler/purge_test.go#TestEndToEndCallPurgeDoNothing @test:server/handler/purge_test.go#TestEndToEndCallPurgeWithRedisDown
  Scenario Outline: A purge says whether nothing was cached or the purge failed
    Given <state>
    When an authorised operator sends PURGE /cart
    Then the response status is <status>
    And the response body is "KO"

    Examples:
      | state                             | status |
      | nothing is cached for /cart       | 404    |
      | the cache store cannot be reached | 500    |
