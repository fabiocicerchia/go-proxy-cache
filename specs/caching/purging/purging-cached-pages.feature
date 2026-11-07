@domain:caching @area:purging @from:cache/cache
Feature: Purging cached pages
  Lets operators remove stale content from the proxy for one page or a whole host, without touching other hosts' content.

  Background:
    Given the operator holds the site's purge key

  @id:caching.purging.001 @rule:caching.purge.escaped-star-is-single-page @verified @source:server/storage/storage.go#PurgeCachedContent @test:cache/purge_host_test.go#TestPurgeHostDeletesOnlyThatHost
  Scenario Outline: An encoded star or a query string purges a single page, not the host
    Given several pages of https://a.example are stored
    When the operator sends PURGE https://a.example<target>
    Then only the page at that exact address is removed
    And the other pages of https://a.example remain stored

    Examples:
      | target |
      | /%2A   |
      | /*?v=1 |
