@domain:operations @area:server-lifecycle @from:server/listeners
Feature: Restarting without refusing connections
  Lets operators reload configuration or binary, stop the service, or hand its ports to systemd without clients seeing refused connections or cut-off responses.

  Background:
    Given go-proxy-cache is serving HTTP traffic for example.com

  @id:operations.server-lifecycle.007 @rule:operations.restart.sighup-keeps-connections @verified @source:server/listeners.go#reexec @test:server/listeners_test.go#TestInheritedListenerSurvivesReexec
  Scenario: A SIGHUP restart serves every request without refusing a connection
    Given a client request to example.com is still being answered
    And the operator has updated the configuration on disk
    When the operator sends SIGHUP to the proxy
    Then the in-flight request receives its full response
    And connections arriving during the restart are queued and answered, none is refused
    And the proxy keeps the same process id
    And new requests are answered by the restarted proxy using the configuration on disk

  @id:operations.server-lifecycle.009 @rule:operations.socket-activation.inherited-sockets-used-by-port @verified @source:server/listeners.go#inheritedListeners @test:server/listeners_test.go#TestInheritedListenerSurvivesReexec @test:server/listeners_test.go#TestInternalsReuseTheInheritedListener
  Scenario Outline: Sockets handed over by systemd are used for the listener on the same port
    Given systemd passes the proxy a listening socket on port <port>
    And the proxy is configured with a <listener> listener on port <port>
    When the operator starts the proxy
    Then the <listener> listener answers on the socket systemd passed
    And configured ports without a passed socket are opened by the proxy itself
    And the log shows "Using the inherited listener on port <port>"

    Examples:
      | port  | listener                         |
      | 80    | public HTTP                      |
      | 443   | public HTTPS                     |
      | 52021 | internal healthcheck and metrics |

  @id:operations.server-lifecycle.010 @rule:operations.socket-activation.foreign-pid-ignored @verified @source:server/listeners.go#inheritedFDs @test:server/listeners_test.go#TestInheritedListenerSurvivesReexec
  Scenario Outline: Sockets meant for another process are ignored
    Given systemd passes the proxy a listening socket on port 80
    And LISTEN_PID is <listen_pid>
    When the operator starts the proxy
    Then the socket systemd passed <outcome>

    Examples:
      | listen_pid                 | outcome                         |
      | the proxy's own process id | is used to serve port 80        |
      | not set                    | is used to serve port 80        |
      | another process id         | is ignored                      |

  @id:operations.server-lifecycle.011 @rule:operations.socket-activation.invalid-socket-refuses-start @inferred @source:server/listeners.go#inheritedListeners
  Scenario Outline: The proxy refuses to start when a passed descriptor is not a TCP listening socket
    Given the proxy is passed descriptor 3 that is <kind>
    When the operator starts the proxy
    Then the proxy exits without serving
    And the error reads "<message>"

    Examples:
      | kind                           | message                                       |
      | a UNIX domain listening socket | inherited fd 3 is not a TCP listener          |
      | a regular file                 | inherited fd 3 is not a listening socket: ... |
