@domain:observability @area:logging @from:logger/log
Feature: Request access logging
  Operators get one access-log line per request, in a format they configure, so they can audit traffic and see how each response was served without exposing the log to forged entries.

  Background:
    Given the operator has configured a log format and a time format

  @id:observability.logging.001 @rule:observability.access-log.format-placeholders-substituted @verified @source:logger/log.go#LogRequest @test:logger/log_test.go#TestLogRequest
  Scenario Outline: Each request is logged in the operator's configured format
    Given the log format is "$host - $remote_addr - $remote_user $protocol $request_method \"$request\" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\""
    When a client requests <path> on <host> from <remote_addr> with referer <referer> and user agent <agent> and receives <status> with a <bytes>-byte body
    Then the access log gets the line "<host> - <remote_addr> - - <protocol> <method> \"<path>\" <status> <bytes> \"<referer>\" \"<agent>\""
    And the line is logged at info level and tagged with the request ID
    And $remote_user is always written as "-"
    And $time_local is written as the local time in the configured time format

    Examples:
      | host             | remote_addr  | protocol | method | path          | status | bytes | referer                   | agent        |
      | example.org      | 127.0.0.1    | HTTP/1.1 | GET    | /path/to/file | 404    | 7     | https://www.example.com   | GoProxyCache |
      | shop.example.com | 203.0.113.10 | HTTP/2.0 | POST   | /cart         | 200    | 512   | https://shop.example.com/ | Mozilla/5.0  |

  @id:observability.logging.002 @rule:observability.access-log.unknown-protocol-method-placeholder @verified @source:logger/log.go#LogRequest @test:logger/log_test.go#TestLogRequest
  Scenario: A missing protocol or method is logged as a question mark
    Given the log format includes $protocol and $request_method
    When a request with no protocol and no method is logged
    Then the access log line shows "?" where the protocol goes
    And the access log line shows "?" where the method goes

  @id:observability.logging.003 @rule:observability.access-log.client-values-cannot-forge-lines @inferred @source:logger/log.go#LogRequest
  Scenario Outline: Client-supplied values cannot add extra lines to the access log
    Given the log format includes $request, $http_user_agent and $http_referer
    When a client sends a request whose <field> contains a carriage return and a line feed followed by fake log text
    Then the request produces exactly one access log line
    And the carriage return and line feed are removed and the fake text stays on that same line

    Examples:
      | field             |
      | URL               |
      | User-Agent header |
      | Referer header    |
      | Host header       |

  @id:observability.logging.004 @rule:observability.access-log.cache-status-fields @verified @source:logger/log.go#LogRequest @test:logger/log_test.go#TestLogRequest @questionable
  Scenario Outline: The access log reports how each response was served
    Given the log format includes $cached_status and $cached_status_label
    When a request is served with cache outcome <outcome>
    Then $cached_status is written as <cached_status>
    And $cached_status_label is written as <label>

    Examples:
      | outcome                                                   | cached_status | label |
      | hit                                                       | true          | HIT   |
      | miss                                                      | false         | MISS  |
      | stale                                                     | true          | STALE |
      | not applicable (purge, refused CONNECT method)            | true          | -     |
