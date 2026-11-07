@domain:operations @area:server-lifecycle @from:go-proxy-cache/go-proxy-cache
Feature: Starting the proxy from the command line
  Operators start, check and inspect go-proxy-cache with command-line flags, so they can confirm a config is valid and see which version runs before they put it in front of live traffic.

  Background:
    Given the operator has the go-proxy-cache binary installed

  @id:operations.server-lifecycle.001 @rule:operations.cli.version-flag-prints-version-and-exits @inferred @source:main.go#printVersion
  Scenario: Asking for the version prints it and does not start the proxy
    Given go-proxy-cache version 1.7.0 is installed
    When the operator runs go-proxy-cache with -version
    Then "1.7.0" is printed on standard output
    And the process exits with status 0
    And no listener is opened

  @id:operations.server-lifecycle.002 @rule:operations.cli.config-test-accepts-valid-file @verified @source:main.go#testConfiguration @test:config/override_ttl_test.go#TestOverrideTTLValidYAML
  Scenario: Checking a valid configuration reports it as valid without starting the proxy
    Given config.yml is well-formed and sets cache override_ttl to 300
    When the operator runs go-proxy-cache with -test -config config.yml
    Then "Configuration file valid." is printed
    And the process exits with status 0
    And no listener is opened

  @id:operations.server-lifecycle.003 @rule:operations.cli.config-test-rejects-invalid-file @verified @source:main.go#testConfiguration @test:config/override_ttl_test.go#TestOverrideTTLNegativeRejected @questionable
  Scenario Outline: Checking an invalid configuration reports it as not valid, with the reason
    Given <file> <problem>
    When the operator runs go-proxy-cache with -test -config <file>
    Then "Configuration file not valid." is printed
    And the next line names the reason, containing "<reason>"
    And the process exits with status 1

    Examples:
      | file        | problem                                            | reason                    |
      | config.yml  | sets cache override_ttl to -1                      | override_ttl must be >= 0 |
      | config.yml  | sets override_ttl to -1 for the domain example_com | override_ttl must be >= 0 |
      | config.yml  | contains an unknown key "cahce"                    | not found in type         |
      | missing.yml | does not exist                                     | no such file or directory |

  @id:operations.server-lifecycle.004 @rule:operations.cli.default-config-file-is-config-yml @inferred @source:main.go#initFlags
  Scenario: Without a config flag the proxy reads config.yml from the working directory
    Given the working directory contains a valid config.yml
    When the operator runs go-proxy-cache with -test and no -config flag
    Then config.yml in the working directory is the file that gets checked
    And "Configuration file valid." is printed

  @id:operations.server-lifecycle.005 @rule:operations.cli.log-flag-redirects-logs-to-file @inferred @source:logger/log.go#InitLogs
  Scenario: Logs go to the file named by the log flag instead of standard output
    Given /var/log/gpc.log is writable
    When the operator starts go-proxy-cache with -log /var/log/gpc.log
    Then log lines are appended to /var/log/gpc.log
    And log lines no longer appear on standard output
    And the file is created readable and writable only by its owner if it did not exist

  @id:operations.server-lifecycle.006 @rule:operations.cli.debug-flag-shows-start-up-diagnostics @inferred @source:main.go#main
  Scenario: Debug mode shows the start-up banner and runtime details
    Given the operator has a valid config.yml
    When the operator starts go-proxy-cache with -debug
    Then the log shows the go-proxy-cache banner and copyright line
    And the log shows the version, Go version, thread count, OS and architecture
    And without -debug none of these lines are logged
