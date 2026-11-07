@domain:caching @area:freshness @from:server/handler
Feature: Reporting and bypassing the cache
  Lets clients and operators see whether a response came from the cache, and fetch fresh content when they need it.

  Background:
    Given the domain "shop.example.com" caches GET responses

  @id:caching.freshness.001 @rule:caching.status.reported-per-response @verified @source:server/handler/http.go#HandleHTTPRequestAndProxy @test:server/handler/purge_test.go#TestEndToEndCallPurge
  Scenario Outline: Every proxied page says whether it was a hit, a stale hit or a miss
    Given the page /cart is <state>
    When a client sends GET /cart
    Then the response carries X-Go-Proxy-Cache-Status "<status>"

    Examples:
      | state                         | status |
      | not cached                    | MISS   |
      | cached and fresh              | HIT    |
      | cached but past its freshness | STALE  |

  @id:caching.freshness.002 @rule:caching.status.force-fresh-bypasses @verified @source:server/handler/http.go#HandleHTTPRequestAndProxy @test:test/etag_test.go#TestETagValidResponse @questionable
  Scenario: A client can demand fresh content, skipping the cached copy
    Given the page /cart is cached and fresh
    When a client sends GET /cart with X-Go-Proxy-Cache-Force-Fresh "1"
    Then the page is fetched from the upstream
    And the response carries X-Go-Proxy-Cache-Status "MISS"
