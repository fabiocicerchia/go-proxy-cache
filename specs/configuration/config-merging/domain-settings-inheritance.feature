@domain:configuration @area:config-merging @from:config/merge
Feature: Inheriting global settings per domain
  Lets an operator hosting several domains define shared settings once and override only what differs per domain, so each domain gets a complete, predictable configuration.

  Background:
    Given the operator's config file defines global settings and a "domains" section

  @id:configuration.config-merging.004 @rule:configuration.merging.domain-inherits-unset-settings @verified @source:config/merge.go#CopyOverWith @test:config/merge_test.go#TestYAMLMergeKeepsCacheFields @test:config/override_ttl_test.go#TestOverrideTTLMergesPerDomain @test:config/merge_test.go#TestCopyOverWithIsTotal
  Scenario Outline: A domain inherits every global setting it does not set
    Given the global settings set <setting> to <global value>
    And domain "shop.example.com" sets only its own upstream host
    When the operator starts the proxy with this config file
    Then the effective setting <setting> for "shop.example.com" is <global value>

    Examples:
      | setting                   | global value     |
      | cache eviction_policy     | allkeys-lru      |
      | cache negative_ttl        | 404: 30, 502: 10 |
      | circuit_breaker threshold | 9                |
      | cache override_ttl        | 60               |
      | server purge secret       | s3cr3t-example   |

  @id:configuration.config-merging.005 @rule:configuration.merging.domain-value-overrides-global @verified @source:config/merge.go#CopyOverWith @test:config/merge_test.go#TestYAMLMergeKeepsCacheFields @test:config/override_ttl_test.go#TestOverrideTTLMergesPerDomain @test:config/merge_test.go#TestCopyOverWithIsTotal
  Scenario Outline: A setting a domain defines replaces the global value for that domain only
    Given the global settings set <setting> to <global value>
    And domain "shop.example.com" sets <setting> to <domain value>
    When the operator starts the proxy with this config file
    Then the effective setting <setting> for "shop.example.com" is <domain value>
    And the global setting <setting> is still <global value>

    Examples:
      | setting               | global value   | domain value     |
      | cache eviction_policy | allkeys-lru    | volatile-ttl     |
      | tls hsts enabled      | false          | true             |
      | cache override_ttl    | 60             | 300              |
      | server upstream host  | global.example | shop.example.com |

  @id:configuration.config-merging.006 @rule:configuration.merging.negative-ttl-replaced-whole @verified @source:config/merge.go#copyOverWithCache @test:config/merge_test.go#TestYAMLMergeKeepsCacheFields
  Scenario Outline: A domain's negative TTL list replaces the global list as a whole
    Given the global negative_ttl is "404: 30, 502: 10"
    And domain "shop.example.com" sets negative_ttl to <domain negative_ttl>
    When the operator starts the proxy with this config file
    Then the effective negative_ttl for "shop.example.com" is <effective negative_ttl>

    Examples:
      | domain negative_ttl | effective negative_ttl        |
      | 404: 5              | 404: 5 (502 is not inherited) |
      | {}                  | empty (no negative caching)   |

  @id:configuration.config-merging.007 @rule:configuration.merging.zero-value-cannot-override @verified @source:utils/utils.go#Coalesce @test:config/merge_test.go#TestCopyOverWithIsTotal @questionable
  Scenario Outline: A domain cannot switch off, zero or blank a setting that is set globally
    Given the global settings set <setting> to <global value>
    And domain "shop.example.com" sets <setting> to <domain value>
    When the operator starts the proxy with this config file
    Then the effective setting <setting> for "shop.example.com" is <global value>

    Examples:
      | setting             | global value   | domain value |
      | tls hsts enabled    | true           | false        |
      | server gzip         | true           | false        |
      | cache ttl           | 3600           | 0            |
      | server purge secret | s3cr3t-example | ""           |
      | tracing enabled     | true           | false        |

  @id:configuration.config-merging.008 @rule:configuration.merging.get-head-always-cacheable @inferred @source:config/merge.go#copyOverWithCache
  Scenario Outline: GET and HEAD are always cacheable methods, whatever the operator lists
    Given the global settings set the cache allowed_methods to <configured methods>
    When the operator starts the proxy with this config file
    Then the effective allowed_methods are <effective methods>

    Examples:
      | configured methods | effective methods |
      | POST               | POST, HEAD, GET   |
      | (not set)          | HEAD, GET         |
      | GET, PURGE         | GET, PURGE, HEAD  |
