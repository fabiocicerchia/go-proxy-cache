@domain:configuration @area:multi-domain @from:config/domain
Feature: Serving several domains from one proxy
  Operators host several sites behind one proxy, and every request has to get its own domain's settings, or be refused when no domain matches it.

  Background:
    Given the operator's global settings have upstream host "global.example.com" over "http"
    And the operator has configured a domain "shop" with upstream host "shop.example.com" over "https"
    And the operator has configured a domain "shop-plain" with upstream host "shop.example.com" over "http"
    And the operator has configured a domain "blog" with upstream host "blog.example.com" over "https"

  @id:configuration.multi-domain.001 @rule:config.domains.request-matched-to-domain-settings @verified @source:config/domain.go#domainConfLookup @test:server/storage/storage_test.go#TestStoreGeneratedPageNegativeTTLFromYAML
  Scenario Outline: A request gets the settings of the domain matching its host, preferring the one that also matches its scheme
    Given the proxy is running with these domains
    When a client sends a request for host "<host>" over "<scheme>"
    Then the request is served with the settings of "<settings>"

    Examples:
      | host               | scheme | settings            |
      | shop.example.com   | https  | shop                |
      | shop.example.com   | http   | shop-plain          |
      | blog.example.com   | http   | blog                |
      | global.example.com | http   | the global settings |
      | global.example.com | https  | the global settings |

  @id:configuration.multi-domain.002 @rule:config.domains.unknown-host-refused @inferred @source:config/domain.go#domainConfLookup
  Scenario: A request for a host no domain is configured for is refused
    Given no domain or global setting has upstream host "unknown.example.net"
    When a client sends a request for host "unknown.example.net" over "https"
    Then the client receives status 501 Not Implemented
    And the proxy logs that no domain in the config matches host "unknown.example.net" with scheme "https"

  @id:configuration.multi-domain.003 @rule:config.domains.every-domain-served-at-startup @inferred @source:config/domain.go#GetDomains
  Scenario: Every configured domain and the global upstream is served once the proxy starts
    Given the proxy is not running yet
    When the operator starts the proxy
    Then the proxy serves "global.example.com", "shop.example.com" and "blog.example.com"
    And each distinct host and scheme pair is set up once, even when two settings repeat it
