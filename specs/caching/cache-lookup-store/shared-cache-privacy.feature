@domain:caching @area:cache-lookup-store @from:server/storage
Feature: Keeping private responses out of the shared cache
  Protects visitors' personal content from being served to other visitors, whatever lifetime the operator configures.

  Background:
    Given the operator has configured a domain that proxies to an origin
    And the domain allows <status> responses to be kept

  @id:caching.cache-lookup-store.002 @rule:caching.store.private-responses-never-shared @verified @source:server/storage/storage.go#StoreGeneratedPage @test:server/storage/storage_test.go#TestStoreGeneratedPageOverrideTTLDoesNotStoreForbidden @test:server/storage/storage_test.go#TestStoreGeneratedPageNegativeTTLDoesNotStorePrivate
  Scenario Outline: Configured lifetimes never make a private or personalised response shareable
    Given the domain has <lifetime_setting>
    And the visitor's request carries "<request_headers>"
    And the origin answers <status> with "<response_headers>"
    When the visitor requests the page
    Then the response is not kept for other visitors
    And the next visitor's request is forwarded to the origin

    Examples:
      | lifetime_setting     | request_headers         | status | response_headers                   |
      | override_ttl 300     | (none)                  | 200    | Cache-Control: private, max-age=60 |
      | override_ttl 300     | (none)                  | 200    | Cache-Control: no-store            |
      | override_ttl 300     | (none)                  | 200    | Cache-Control: no-cache            |
      | override_ttl 300     | Authorization: Bearer x | 200    | Cache-Control: max-age=60          |
      | override_ttl 300     | Authorization: Bearer x | 200    | (none)                             |
      | negative_ttl 404: 30 | (none)                  | 404    | Cache-Control: private             |
