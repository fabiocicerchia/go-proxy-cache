@domain:observability @area:logging @from:utils/utils
Feature: Tamper-proof request logs
  Operators who rely on the proxy's logs need every logged request to stay on one line, so a client cannot forge log entries.

  Background:
    Given the proxy is serving traffic for www.example.com

  @id:observability.logging.010 @rule:observability.logging.request-values-cannot-forge-log-lines @verified @source:utils/utils.go#EscapeLogValue @test:utils/utils_test.go#TestEscapeLogValueStripsCRLF @test:utils/utils_test.go#TestEscapeLogValueLeavesCleanValueAlone
  Scenario Outline: Line breaks sent by a client never split a log entry
    Given the access log records the request line, host, user agent and referer
    When a client sends a request whose <field> is "<sent>"
    Then the log entry for that request is a single line
    And the logged <field> reads "<logged>"

    Examples:
      | field        | sent                              | logged                        |
      | request path | /path\r\nINFO forged log line     | /pathINFO forged log line     |
      | user agent   | curl/8.0\nINFO admin login ok     | curl/8.0INFO admin login ok   |
      | referer      | https://example.com/\r\nWARN fake | https://example.com/WARN fake |
      | request path | /path?a=b                         | /path?a=b                     |
