@domain:caching @area:cache-backend @from:cache/engine
Feature: Running the cache on Redis
  Lets operators run the shared Redis cache store with a memory-eviction policy they choose, and makes sure purges and stored responses act predictably when Redis is down or a response has no Vary header.

  Background:
    Given the proxy is configured with a Redis cache store for the domain example.com

  @id:caching.cache-backend.001 @rule:caching.cache-backend.eviction-policy-allowlist @inferred @source:cache/engine/client/client.go#applyEvictionPolicy @questionable
  Scenario Outline: Only a recognised eviction policy is applied to the cache store
    Given the operator sets the cache eviction policy to "<policy>"
    When the operator starts the proxy
    Then the cache store's memory eviction policy is <result>
    And the proxy logs <log>

    Examples:
      | policy        | result                          | log                                                     |
      | allkeys-lru   | allkeys-lru                     | nothing about the eviction policy                       |
      | volatile-ttl  | volatile-ttl                    | nothing about the eviction policy                       |
      |               | left at the store's own default | nothing about the eviction policy                       |
      | lru; FLUSHALL | left at the store's own default | Invalid REDIS_EVICTION_POLICY "lru; FLUSHALL", skipping |

  @id:caching.cache-backend.002 @rule:caching.purging.store-unreachable-is-failure @verified @source:cache/engine/client/client_wildcard.go#DelWildcard @test:server/handler/purge_test.go#TestEndToEndCallPurgeWithRedisDown
  Scenario: A purge that cannot reach the cache store is reported as a failure, not as nothing to purge
    Given the cache store for example.com is unreachable
    And the client has a valid purge secret
    When the client sends PURGE for https://example.com/
    Then the response status is 500
    And the response body is "KO"

  @id:caching.cache-backend.003 @rule:caching.cache-lookup-store.store-without-vary @verified @source:cache/engine/client/client.go#Push @test:cache/engine/client/client_test.go#TestPushListEmptyIsANoOp
  Scenario: A response without a Vary header is still cached
    Given the upstream answers GET https://example.com/page with status 200, a cacheable lifetime and no Vary header
    When a visitor requests https://example.com/page
    Then the response is stored for later visitors
    And the stored copy has no variants listed
