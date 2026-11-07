@domain:observability @area:logging @from:logger/log
Feature: Choosing log destinations and verbosity
  Operators pick where proxy logs go (stdout, a file, Sentry, syslog) and how detailed they are, so they can plug the proxy into their monitoring.

  @id:observability.logging.005 @rule:observability.log-output.log-file-replaces-stdout @inferred @source:logger/log.go#InitLogs
  Scenario: With a log file set, logs are appended to that file instead of stdout
    Given the file /var/log/proxy.log already holds earlier log lines
    When the operator starts the proxy with --log /var/log/proxy.log
    Then new log lines are appended after the existing ones in /var/log/proxy.log
    And nothing is logged to stdout
    And a missing file is created, readable and writable only by its owner (mode 0600)
    And the proxy stops at start-up if the file cannot be opened

  @id:observability.logging.006 @rule:observability.log-output.debug-flag-enables-debug-logs @inferred @source:logger/log.go#SetDebugLevel
  Scenario Outline: The debug flag turns on debug-level logs
    When the operator starts the proxy <with_debug>
    Then debug messages such as the start-up banner are <shown>
    And info and higher messages are logged

    Examples:
      | with_debug      | shown      |
      | with --debug    | logged     |
      | without --debug | not logged |

  @id:observability.logging.007 @rule:observability.log-output.sentry-receives-warnings-and-above @inferred @source:logger/log.go#HookSentry
  Scenario Outline: Warnings and errors go to Sentry when a Sentry DSN is configured
    Given the operator has set log.sentry_dsn to a valid DSN
    When the proxy logs a message at <level>
    Then the message is <sent> to Sentry, with a stack trace

    Examples:
      | level   | sent     |
      | warning | sent     |
      | error   | sent     |
      | fatal   | sent     |
      | panic   | sent     |
      | info    | not sent |
      | debug   | not sent |

  @id:observability.logging.008 @rule:observability.log-output.syslog-requires-endpoint @inferred @source:logger/log.go#HookSyslog @questionable
  Scenario Outline: Logs go to syslog only when a syslog endpoint is configured
    Given the operator has set log.syslog_protocol to <protocol> and log.syslog_endpoint to <endpoint>
    When the proxy starts and logs a warning
    Then the warning is <outcome>

    Examples:
      | protocol | endpoint               | outcome                                         |
      | udp      | syslog.example.com:514 | sent to syslog.example.com:514                  |
      | (empty)  | (empty)                | not sent to any syslog, including the local one |

  @id:observability.logging.009 @rule:observability.log-output.startup-summary @verified @source:logger/log.go#LogSetup @test:logger/log_test.go#TestLogSetup @test:logger/log_test.go#TestLogSetupWithoutEndpoints
  Scenario Outline: At start-up the proxy logs its ports and upstream target
    Given a server listens on HTTP port 80 and HTTPS port 443
    And its upstream is https://www.example.com with endpoints <endpoints>
    When the proxy starts
    Then the log shows "Server will run on :80 and :443 and redirects to url: https://www.example.com -> <shown>"
    And an empty upstream host is shown as "*"

    Examples:
      | endpoints            | shown                 |
      | 192.0.2.4, 192.0.2.8 | [192.0.2.4 192.0.2.8] |
      | none                 | VOID                  |
