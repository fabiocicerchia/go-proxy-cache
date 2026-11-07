@domain:proxying @area:request-forwarding @from:server/server
Feature: Limiting how long a request may take
  Operators cap request time so slow clients or upstreams cannot tie up the proxy, and clients get a clear answer when the cap is hit.

  @id:proxying.request-forwarding.005 @rule:proxying.timeouts.slow-request-gets-503 @gap @source:server/server.go#InitServer
  Scenario: A request that exceeds the handler timeout is answered with Timed Out
    Given the handler timeout is 5s
    And the upstream for www.example.com takes 10 seconds to answer
    When a client requests https://www.example.com/report
    Then the client receives status 503 after about 5 seconds
    And the response body is "Timed Out" followed by a newline

  @id:proxying.request-forwarding.006 @rule:proxying.timeouts.global-for-all-domains @gap @source:server/server.go#InitServer @questionable
  Scenario: Server timeouts are the same for every domain, whatever a domain sets
    Given the global handler timeout is 5s
    And the domain api.example.com sets its own handler timeout of 30s
    When a client request to api.example.com takes 10 seconds upstream
    Then the client receives status 503 with body "Timed Out" (plus a newline) after about 5 seconds

  @id:proxying.request-forwarding.007 @rule:configuration.timeouts.bare-numbers-are-seconds @gap @source:server/server.go#normalizeTimeout @questionable
  Scenario Outline: Server timeouts written as bare numbers are read as seconds
    Given the config file sets the <setting> timeout to <value>
    When the operator starts the proxy
    Then the <setting> timeout in effect is <effective>

    Examples:
      | setting  | value | effective                                               |
      | read     | 5     | 5 seconds                                               |
      | write    | 5s    | 5 seconds                                               |
      | idle     | 20    | 20 seconds                                              |
      | shutdown | 10    | 10 seconds                                              |
      | handler  | 5     | 5 nanoseconds, so every request is answered "Timed Out" |
