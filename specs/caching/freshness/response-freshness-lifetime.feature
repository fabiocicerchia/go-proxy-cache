@domain:caching @area:freshness @from:utils/ttl
Feature: Deciding how long a proxied response stays fresh
  Lets operators and origin owners control, through standard HTTP caching headers and a per-domain default, how long the proxy keeps serving a response without going back to the origin.

  Background:
    Given the domain's default cache TTL is 3600 seconds
    And no override TTL or negative TTL applies to the response

  @id:caching.freshness.006 @rule:caching.freshness.default-ttl-without-origin-freshness @verified @source:utils/ttl/ttl.go#GetTTL @test:utils/ttl/ttl_test.go#TestGetTTLWhenNotSet @test:utils/ttl/ttl_test.go#TestGetTTLWhenSetCacheControlEmpty @test:utils/ttl/ttl_test.go#TestGetTTLWhenInvalidExpires
  Scenario Outline: A response without freshness information uses the domain's default lifetime
    Given the origin sends no usable s-maxage, max-age or Expires
    When the origin responds with headers <headers>
    Then the response stays fresh for 3600 seconds

    Examples:
      | headers               |
      | (none)                |
      | Cache-Control: public |
      | Expires: test         |

  @id:caching.freshness.007 @rule:caching.freshness.shared-max-age-precedence @verified @source:utils/ttl/ttl.go#ttlFromCacheControlChain @test:utils/ttl/ttl_test.go#TestGetTTLWhenSetCacheControlSmaxage @test:utils/ttl/ttl_test.go#TestGetTTLWhenSetCacheControlMaxage @test:utils/ttl/ttl_test.go#TestGetTTLSmaxageWinsOverMaxage @test:utils/ttl/ttl_test.go#TestGetTTLWhenMaxageZero @test:utils/ttl/ttl_test.go#TestGetTTLWhenSetCacheControlAndExpires
  Scenario Outline: The shared-cache lifetime from the origin wins over the browser lifetime and Expires
    When the origin responds with headers <headers>
    Then the response stays fresh for <seconds> seconds

    Examples:
      | headers                                                                          | seconds |
      | Cache-Control: public, max-age=3600, s-maxage=86400                              | 86400   |
      | Cache-Control: s-maxage=0, max-age=60                                            | 0       |
      | Cache-Control: public, max-age=120                                               | 120     |
      | Cache-Control: public, max-age=0                                                 | 0       |
      | Cache-Control: public, max-age=3600, s-maxage=86400 and Expires 60 seconds ahead | 86400   |

  @id:caching.freshness.008 @rule:caching.freshness.expires-sets-lifetime @verified @source:utils/ttl/ttl.go#overrideWithExpires @test:utils/ttl/ttl_test.go#TestGetTTLWhenSetExpires
  Scenario: A future Expires date sets the lifetime to the time remaining until it
    Given the origin sends no s-maxage or max-age
    When the origin responds with an Expires date 60 seconds in the future
    Then the response stays fresh for just under 60 seconds

  @id:caching.freshness.009 @rule:caching.freshness.past-expires-not-treated-as-stale @inferred @source:utils/ttl/ttl.go#ttlFromExpires @questionable
  Scenario Outline: An Expires date already in the past falls back to the domain's default lifetime
    Given the origin sends no s-maxage or max-age
    When the origin responds with an Expires date of <expires>
    Then the response stays fresh for 3600 seconds

    Examples:
      | expires                       |
      | Thu, 01 Jan 1970 00:00:00 GMT |
      | 0                             |

  @id:caching.freshness.010 @rule:caching.freshness.invalid-max-age-is-stale @inferred @source:utils/ttl/ttl.go#parseDeltaSeconds
  Scenario Outline: An invalid or negative max-age makes the response immediately stale
    When the origin responds with Cache-Control: <cache_control>
    Then the response stays fresh for 0 seconds

    Examples:
      | cache_control            |
      | max-age=-5               |
      | max-age=soon             |
      | s-maxage=abc, max-age=60 |

  @id:caching.freshness.011 @rule:caching.freshness.private-responses-never-shared @verified @source:utils/ttl/ttl.go#ForbidsStoring @test:utils/ttl/ttl_test.go#TestGetTTLWhenCacheControlNoCache @test:utils/ttl/ttl_test.go#TestGetTTLWhenCacheControlNoStore @test:utils/ttl/ttl_test.go#TestGetTTLWhenCacheControlPrivateWithMaxage @test:utils/ttl/ttl_test.go#TestGetTTLWhenCacheControlPrivateWithoutMaxage @test:utils/ttl/ttl_test.go#TestGetTTLWhenCacheControlQualifiedPrivate @test:utils/ttl/ttl_test.go#TestGetTTLParsesCaseWhitespaceAndQuotes
  Scenario Outline: Responses the origin marks no-store, private or no-cache are never kept for other clients
    When the origin responds with Cache-Control: <cache_control>
    Then the response stays fresh for 0 seconds
    And it is not stored for later clients

    Examples:
      | cache_control                           |
      | private, no-store, max-age=3600         |
      | private, no-cache, max-age=3600         |
      | private, max-age=60                     |
      | private                                 |
      | private="Set-Cookie, X-Foo", max-age=60 |
      | No-Store                                |

  @id:caching.freshness.012 @rule:caching.freshness.cache-control-read-as-whole-tokens @verified @source:utils/ttl/ttl.go#ParseCacheControl @test:utils/ttl/ttl_test.go#TestGetTTLParsesCaseWhitespaceAndQuotes @test:utils/ttl/ttl_test.go#TestGetTTLCombinesMultipleCacheControlHeaders @test:utils/ttl/ttl_test.go#TestGetTTLMatchesDirectivesAsTokens
  Scenario Outline: Cache-Control directives are honoured whatever their case, spacing, quoting or number of header lines
    When the origin responds with Cache-Control <cache_control>
    Then the response stays fresh for <seconds> seconds

    Examples:
      | cache_control                                   | seconds |
      |  PUBLIC ,  MAX-AGE="120"                        | 120     |
      | "public" and "max-age=120" on two header lines  | 120     |
      | "max-age=120" and "private" on two header lines | 0       |
      | x-no-store-hint, x-private-ish, max-age=60      | 60      |
