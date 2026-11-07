@domain:configuration @area:config-loading @from:config/config
Feature: Loading and checking the configuration
  Lets operators configure the proxy from a YAML file and environment variables, and check a config file before deploying it, so a bad setting is caught instead of silently served.

  Background:
    Given the proxy is started with a config file path (default config.yml)

  @id:configuration.config-loading.001 @rule:configuration.loading.test-flag-reports-validity @verified @source:config/config.go#Validate @test:config/override_ttl_test.go#TestOverrideTTLNegativeRejected @test:config/override_ttl_test.go#TestOverrideTTLValidYAML
  Scenario Outline: Checking a config file reports whether it is valid
    Given a config file whose content is <content>
    When the operator runs the proxy with --test against that file
    Then the output says "<message>"
    And the command exits with code <exit>

    Examples:
      | content                                       | message                       | exit |
      | cache: override_ttl: 300                      | Configuration file valid.     | 0    |
      | an unknown key such as cache: colour: blue    | Configuration file not valid. | 1    |
      | cache: override_ttl: -1                       | Configuration file not valid. | 1    |
      | domains: example_com: cache: override_ttl: -1 | Configuration file not valid. | 1    |
      | a path to a file that does not exist          | Configuration file not valid. | 1    |

  @id:configuration.config-loading.002 @rule:configuration.loading.runs-from-environment-without-file @verified @source:config/config.go#InitConfigFromFileOrEnv @test:config/override_ttl_test.go#TestOverrideTTLFromEnv
  Scenario: The proxy runs on environment settings alone when no config file exists
    Given no file exists at the config file path
    And the environment sets OVERRIDE_TTL to 120
    When the operator starts the proxy
    Then the proxy starts without error
    And the forced cache TTL in effect is 120 seconds

  @id:configuration.config-loading.003 @rule:configuration.loading.file-overrides-environment @inferred @source:config/config.go#InitConfigFromFileOrEnv @questionable
  Scenario: A setting in the config file takes precedence over the same environment variable
    Given the environment sets OVERRIDE_TTL to 120
    And the config file sets cache override_ttl to 300
    When the operator starts the proxy
    Then the forced cache TTL in effect is 300 seconds

  @id:configuration.config-loading.004 @rule:configuration.loading.invalid-settings-refuse-start @inferred @source:config/config.go#loadYAMLFile
  Scenario Outline: The proxy refuses to start with invalid settings
    Given <source> contains <problem>
    When the operator starts the proxy
    Then the proxy logs a fatal error naming the problem
    And the proxy does not start serving

    Examples:
      | source          | problem                                              |
      | the config file | an unknown key                                       |
      | the config file | a negative cache override_ttl for domain example_com |
      | the environment | OVERRIDE_TTL set to -1                               |

  @id:configuration.config-loading.005 @rule:configuration.loading.tls-paths-relative-to-config-file @inferred @source:config/config.go#patchAbsFilePath
  Scenario Outline: Relative certificate paths in the config file are resolved from the file's folder
    Given the config file at /etc/proxy/config.yml sets the TLS cert_file to <path>
    When the operator starts the proxy
    Then the proxy loads the certificate from <resolved>

    Examples:
      | path                | resolved                    |
      | certs/server.crt    | /etc/proxy/certs/server.crt |
      | /opt/tls/server.crt | /opt/tls/server.crt         |
