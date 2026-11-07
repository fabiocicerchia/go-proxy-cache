@domain:configuration @area:config-merging @from:utils/utils
Feature: Domain settings inheriting global settings
  Operators hosting several domains need each domain to inherit the global settings it does not set, and to know which values a domain can override.

  Background:
    Given the global configuration enables gzip, HSTS and a cache TTL of 3600 seconds

  @id:configuration.config-merging.010 @rule:configuration.merging.unset-domain-setting-inherits-global @verified @source:utils/utils.go#Coalesce @test:utils/utils_test.go#TestCoalesceFalse @test:utils/utils_test.go#TestCoalesceFallsBackOnTypedNilPointer @test:config/override_ttl_test.go#TestOverrideTTLMergesPerDomain
  Scenario: A setting a domain leaves out takes the global value
    Given the domain www.example.com sets its cache TTL to 60 seconds and sets nothing else
    When the operator starts the proxy
    Then www.example.com uses a cache TTL of 60 seconds
    And www.example.com keeps gzip and HSTS enabled from the global configuration
    And www.example.com keeps the global TLS settings

  @id:configuration.config-merging.011 @rule:configuration.merging.off-or-zero-cannot-override-global @verified @source:utils/utils.go#IsEmpty @test:utils/utils_test.go#TestIsEmptyTrue @questionable
  Scenario Outline: A domain cannot switch off or zero a setting the global configuration turns on
    Given the domain www.example.com sets <setting> to <domain value>
    When the operator starts the proxy
    Then www.example.com uses <setting> = <effective value>

    Examples:
      | setting      | domain value | effective value |
      | gzip         | false        | true            |
      | HSTS enabled | false        | true            |
      | cache TTL    | 0            | 3600            |
