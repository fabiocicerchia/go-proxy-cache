@domain:caching @area:purging @from:server/storage
Feature: Purging a page or a whole host
  Lets operators invalidate one cached URL or everything cached for a host, without touching other hosts.

  Background:
    Given the operator is allowed to send PURGE requests for https://a.example

  @id:caching.purging.005 @rule:caching.purging.host-wide-purge-scoped-to-host-and-scheme @verified @source:server/storage/storage.go#PurgeCachedContent @test:cache/purge_host_test.go#TestPurgeHostDeletesOnlyThatHost
  Scenario: Purging /* clears every cached page of that host and scheme only
    Given pages of https://a.example are cached for GET and HEAD
    And pages of http://a.example and https://b.example are also cached
    When the operator sends PURGE /* to https://a.example
    Then the response is 200 with body "OK"
    And every cached page of https://a.example is gone
    And the cached pages of http://a.example and https://b.example are still served

  @id:caching.purging.006 @rule:caching.purging.escaped-or-queried-star-is-single-url @verified @source:server/storage/storage.go#PurgeCachedContent @test:cache/purge_host_test.go#TestPurgeHostDeletesOnlyThatHost
  Scenario Outline: An encoded or query-carrying star purges only that one URL
    Given pages of https://a.example are cached
    And no page is cached at the literal URL <target>
    When the operator sends PURGE <target> to https://a.example
    Then the response is 404 with body "KO"
    And the cached pages of https://a.example are still served

    Examples:
      | target |
      | /%2A   |
      | /*?v=1 |

  @id:caching.purging.007 @rule:caching.purging.nothing-to-purge-is-not-found @verified @source:server/storage/storage.go#PurgeCachedContent @test:cache/purge_host_test.go#TestPurgeHostDeletesOnlyThatHost
  Scenario: Purging when nothing is cached answers not found
    Given nothing is cached for https://a.example
    When the operator sends PURGE /* to https://a.example
    Then the response is 404 with body "KO"
