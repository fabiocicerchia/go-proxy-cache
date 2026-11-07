@domain:security @area:listeners @from:server/server
Feature: Opening public and internal ports
  Operators get public HTTP and HTTPS ports for each domain, plus healthcheck and metrics endpoints that stay off the public ports.

  @id:security.listeners.001 @rule:security.listeners.internals-only-on-internal-address @verified @source:server/server.go#AttachInternals @test:server/server_unit_test.go#TestAttachInternalsBindsConfiguredAddress @test:server/server_unit_test.go#TestAttachInternalsIPv6 @test:server/listeners_test.go#TestInternalsListenOnTheConfiguredAddress
  Scenario Outline: Health and metrics endpoints are reachable only on the internal address
    Given the internal listening address is 127.0.0.1 and the internal port is 52021
    And www.example.com is served on public ports 80 and 443
    When a client requests <path> on <target>
    Then <outcome>

    Examples:
      | path         | target             | outcome                                                    |
      | /healthcheck | 127.0.0.1:52021    | the proxy's health report is returned                      |
      | /metrics     | 127.0.0.1:52021    | the Prometheus metrics are returned                        |
      | /healthcheck | www.example.com:80 | the request is proxied to the upstream like any other path |
      | /metrics     | 203.0.113.10:52021 | the connection is refused                                  |

  @id:security.listeners.002 @rule:security.listeners.internal-port-must-not-be-public @verified @source:server/server.go#AttachInternals @test:server/server_unit_test.go#TestAttachInternalsRejectsPublicPortCollision
  Scenario Outline: The proxy refuses to start when the internal port is also a public port
    Given www.example.com is served on public port <public_port>
    And the internal port is configured as <public_port>
    When the operator starts the proxy
    Then the proxy exits without serving
    And the log reports that the internal listening port is already used by a public <kind> listener

    Examples:
      | public_port | kind  |
      | 80          | HTTP  |
      | 443         | HTTPS |

  @id:security.listeners.003 @rule:security.listeners.broken-tls-serves-http-only @gap @source:server/server.go#InitServers @questionable
  Scenario: A domain whose certificate cannot be loaded is served over HTTP only
    Given www.example.com has automatic certificates turned off
    And its certificate or key file is missing or unreadable
    When the operator starts the proxy
    Then www.example.com answers on its HTTP port
    And no HTTPS listener is opened for www.example.com
    And the log reports "No HTTPS server will be listening on 'www.example.com'"
