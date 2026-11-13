@domain:configuration @area:config-merging @from:config/config
Feature: Per-domain settings
  Lets operators host several domains with their own settings while sharing global defaults, including where each domain's token signing keys come from.

  Background:
    Given the config file defines global settings and a domains section

  @id:configuration.config-merging.002 @rule:configuration.domains.defined-only-in-file @inferred @source:config/config.go#InitConfigFromFileOrEnv
  Scenario: Domains can only be defined in the config file
    Given no file exists at the config file path
    When the operator starts the proxy
    Then no per-domain settings are in effect
    And every request uses the global settings

  @id:configuration.config-merging.003 @rule:configuration.domains.jwks-url-env-override @verified @source:config/config.go#copyGlobalOverDomainConfig @test:server/jwt/jwt_test.go#TestJWKSUrlFromYaml @test:server/jwt/jwt_test.go#TestJWKSUrlFromEnvAndYaml @test:server/jwt/jwt_test.go#TestJWKSUrlFromEnv
  Scenario Outline: A domain's token key set URL can be set from the environment
    Given the config file sets jwks_url for domain example_com to http://keys.example.com/yaml
    And <env>
    When the operator starts the proxy
    Then tokens for example_com are verified against keys from <url>

    Examples:
      | env                                                     | url                          |
      | JWT_JWKS_URL_example_com is not set                     | http://keys.example.com/yaml |
      | JWT_JWKS_URL_example_com is http://keys.example.com/env | http://keys.example.com/env  |
