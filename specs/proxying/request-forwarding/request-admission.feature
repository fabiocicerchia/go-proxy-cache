@domain:proxying @area:request-forwarding @from:server/handler
Feature: Accepting and forwarding requests for configured domains
  Lets operators be sure the proxy only serves the hosts and ports they configured, and that upstreams learn who the real client was.

  Background:
    Given the proxy is configured for the domain "shop.example.com" on ports 80 and 443

  @id:proxying.request-forwarding.001 @rule:proxying.admission.unconfigured-host-or-port-refused @verified @source:server/handler/handler.go#initRequestParams @test:server/handler/http_functional_test.go#TestHTTPEndToEndCallWithMissingDomain @test:test/http_test.go#TestHTTPClientCallToMissingDomain
  Scenario Outline: Requests for a host or port the config does not list are refused
    Given no domain in the config matches <host> arriving on port <port>
    When a client sends GET / with Host <host> on port <port>
    Then the response status is 501
    And nothing is forwarded to any upstream
    And the proxy log states the reason: <reason>

    Examples:
      | host                | port | reason                                                     |
      | www.unknown.example | 80   | no domain in the config matches the host                   |
      | shop.example.com    | 8080 | the port is neither server.port.http nor server.port.https |

  @id:proxying.request-forwarding.002 @rule:proxying.admission.connect-method-refused @verified @source:server/handler/handler.go#HandleRequest @test:server/handler/connect_test.go#TestEndToEndCallConnect
  Scenario: The proxy refuses to open tunnels with CONNECT
    When a client sends a CONNECT request for shop.example.com
    Then the response status is 405
    And the response body is empty

  @id:proxying.request-forwarding.003 @rule:proxying.forwarding.client-identity-forwarded @verified @source:server/handler/utils.go#ProxyDirector @test:server/handler/http_unit_test.go#TestXForwardedFor @questionable
  Scenario: The upstream learns the original client, host, scheme and request id
    Given the client at 203.0.113.7 already carries X-Forwarded-For "198.51.100.4"
    When the client sends GET /cart over HTTPS to shop.example.com
    Then the upstream receives X-Forwarded-For "198.51.100.4, 203.0.113.7"
    And the upstream receives X-Forwarded-Proto "https"
    And the upstream receives X-Forwarded-Host with the client's Host header
    And the upstream receives an X-Go-Proxy-Cache-Request-ID header that is unique to the request

  @id:proxying.request-forwarding.004 @rule:proxying.forwarding.upstream-scheme-follows-client @verified @source:server/handler/utils.go#GetUpstreamURL @test:server/handler/utils_test.go#TestGetUpstreamURLWithWildcard
  Scenario Outline: A wildcard upstream scheme follows the client's scheme and its default port
    Given the upstream scheme is configured as "*" with endpoint "origin1" and no port
    When a client sends a request over <client_scheme>
    Then the proxy calls the upstream at <upstream_url>

    Examples:
      | client_scheme | upstream_url        |
      | http          | http://origin1:80   |
      | https         | https://origin1:443 |
