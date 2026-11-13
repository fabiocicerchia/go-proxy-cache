@domain:caching @area:cache-lookup-store @from:cache/cache
Feature: Serving pages from cache
  Lets site operators offload their origin by answering repeat requests from stored pages, while never storing or serving content a shared cache must keep private.

  Background:
    Given the proxy fronts the site www.example.com
    And the site allows GET responses with status 200 to be cached

  @id:caching.cache-lookup-store.001 @rule:caching.lookup.vary-separates-variants @verified @source:cache/cache.go#StorageKey @test:cache/cache_integration_test.go#TestFullPageRoundTripOnRedis @test:cache/cache_unit_test.go#TestVaryHeaderChecksumPreservesNamesAndCase
  Scenario Outline: Visitors who differ in a Vary header get separate stored pages
    Given the origin answers https://www.example.com/page?q=1 with Vary: Accept-Language
    And a visitor with Accept-Language "en" has had that page stored
    When a visitor with Accept-Language "<language>" requests https://www.example.com/page?q=1
    Then the X-Go-Proxy-Cache-Status header is "<status>"

    Examples:
      | language | status |
      | en       | HIT    |
      | it       | MISS   |
