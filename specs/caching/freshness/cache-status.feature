@domain:caching @area:freshness @from:server/handler
Feature: Reporting and bypassing the cache
  Lets clients and operators see whether a response came from the cache, and fetch fresh content when they need it.

  Background:
    Given the domain "shop.example.com" caches GET responses

  @id:caching.freshness.002 @rule:caching.status.force-fresh-bypasses @verified @source:server/handler/http.go#HandleHTTPRequestAndProxy @test:test/etag_test.go#TestETagValidResponse @questionable
  Scenario: A client can demand fresh content, skipping the cached copy
    Given the page /cart is cached and fresh
    When a client sends GET /cart with X-Go-Proxy-Cache-Force-Fresh "1"
    Then the page is fetched from the upstream
    And the response carries X-Go-Proxy-Cache-Status "MISS"
