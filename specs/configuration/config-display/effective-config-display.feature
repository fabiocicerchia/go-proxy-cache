@domain:configuration @area:config-display @from:config/print
Feature: Showing the effective configuration at start-up
  Lets operators see which settings a running proxy actually loaded, without exposing the credentials it holds.

  Background:
    Given go-proxy-cache has loaded its configuration from the config file and environment

  @id:configuration.config-display.001 @rule:configuration.config-display.shown-in-debug-mode @inferred @source:config/print.go#Print @questionable
  Scenario: The effective configuration is not printed at start-up, even in debug mode
    Given the operator has enabled debug logging with the -debug flag
    When the operator starts go-proxy-cache
    Then the start-up log contains no "Config Settings:" section
    And the start-up log does not list the effective configuration values

  @id:configuration.config-display.002 @rule:configuration.config-display.secrets-masked @inferred @source:config/print.go#Print
  Scenario Outline: Credentials are masked in the configuration printout
    Given the <scope> configuration has its <setting> set to "<configured>"
    When go-proxy-cache prints its effective configuration
    Then the printed <setting> for the <scope> configuration reads "<shown>"

    Examples:
      | scope                  | setting        | configured    | shown           |
      | global                 | Redis password | s3cr3t-redis  | *** OMITTED *** |
      | global                 | Redis password |               | *** OMITTED *** |
      | global                 | purge secret   | purge-key-123 | *** OMITTED *** |
      | global                 | purge secret   |               |                 |
      | www.example.com domain | Redis password | s3cr3t-redis  | *** OMITTED *** |
      | www.example.com domain | purge secret   | purge-key-123 | *** OMITTED *** |
      | www.example.com domain | purge secret   |               |                 |

  @id:configuration.config-display.003 @rule:configuration.config-display.sentry-dsn-unmasked @inferred @source:config/print.go#Print @questionable
  Scenario: The Sentry DSN appears unmasked in the configuration printout
    Given the configuration has the Sentry DSN set to "https://abc123@o0.ingest.example.com/42"
    When go-proxy-cache prints its effective configuration
    Then the printout contains "https://abc123@o0.ingest.example.com/42" verbatim
