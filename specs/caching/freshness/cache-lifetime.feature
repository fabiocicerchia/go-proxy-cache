@domain:caching @area:freshness @from:server/storage
Feature: Choosing how long a response stays cached
  Lets operators control how long each origin response is served from the proxy, so broken URLs and badly configured origins do not hammer the upstream.

  Background:
    Given the operator has configured a domain that proxies to an origin

  @id:caching.freshness.004 @rule:caching.freshness.ttl-precedence @verified @source:server/storage/storage.go#StorageTTL @test:server/storage/storage_test.go#TestStorageTTLOverride @test:server/storage/storage_test.go#TestApplyNegativeTTLWithOverride @test:server/storage/storage_test.go#TestApplyNegativeTTLWithoutOverride @test:server/storage/storage_test.go#TestApplyNegativeTTLWithNilMap
  Scenario Outline: A response's cache lifetime follows negative TTL, then override TTL, then origin headers, then the default TTL
    Given the domain's default ttl is <ttl> seconds, override_ttl is <override_ttl> and negative_ttl is <negative_ttl>
    And the origin answers with status <status> and freshness headers "<origin_headers>"
    When a visitor requests the page
    Then the response is kept for <lifetime> seconds

    Examples:
      | ttl | override_ttl | negative_ttl | status | origin_headers             | lifetime |
      | 10  | 0            | none         | 200    | Cache-Control: max-age=60  | 60       |
      | 10  | 0            | none         | 200    | (none)                     | 10       |
      | 10  | 300          | none         | 200    | Cache-Control: max-age=60  | 300      |
      | 0   | 300          | none         | 200    | Cache-Control: s-maxage=60 | 300      |
      | 0   | 300          | none         | 200    | Expires in one hour        | 300      |
      | 0   | 300          | none         | 200    | Cache-Control: max-age=0   | 300      |
      | 0   | 300          | 404: 30      | 404    | Cache-Control: max-age=60  | 30       |
      | 0   | 300          | 404: 30      | 502    | (none)                     | 300      |

  @id:caching.freshness.005 @rule:caching.freshness.negative-ttl-caches-error-pages @verified @source:server/storage/storage.go#StoreGeneratedPage @test:server/storage/storage_test.go#TestStoreGeneratedPageNegativeTTLFromYAML
  Scenario: An error page with no cache headers is still kept for its negative TTL
    Given the config file sets negative_ttl 404 to 30 seconds
    And the domain allows 404 responses to be kept
    And the origin answers 404 Not Found with no cache headers
    When a visitor requests the missing page
    Then the 404 response is kept for 30 seconds instead of being skipped
