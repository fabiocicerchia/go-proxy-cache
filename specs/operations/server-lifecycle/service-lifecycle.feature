@domain:operations @area:server-lifecycle @from:server/server
Feature: Stopping and reloading the proxy without dropping requests
  Operators can stop or reload the proxy while clients keep getting answers, which keeps deploys and config changes from causing outages.

  Background:
    Given the proxy is serving traffic for www.example.com

  @id:operations.server-lifecycle.012 @rule:operations.lifecycle.stop-drains-in-flight-requests @verified @source:server/server.go#shutdownServers @test:server/listeners_test.go#TestShutdownDrainsInFlightRequests
  Scenario Outline: Stopping the proxy lets requests already in progress finish
    Given a client request to www.example.com is still being answered
    And the shutdown timeout is 5s
    When the operator sends <signal> to the proxy
    Then the proxy stops accepting new connections
    And the request in progress receives its full response
    And the proxy exits after the last in-progress request has finished

    Examples:
      | signal  |
      | SIGTERM |
      | SIGINT  |

  @id:operations.server-lifecycle.013 @rule:operations.lifecycle.drain-bounded-by-shutdown-timeout @gap @source:server/server.go#waitForSignal
  Scenario: Requests still running when the shutdown timeout runs out are cut off
    Given a client request to www.example.com takes longer than the shutdown timeout
    And the shutdown timeout is 5s
    When the operator sends SIGTERM to the proxy
    Then the proxy exits about 5 seconds after the signal
    And the slow request does not receive a complete response
    And the log reports that a server could not shut down
