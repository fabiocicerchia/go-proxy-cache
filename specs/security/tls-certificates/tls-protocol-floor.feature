@domain:security @area:tls-certificates @from:config/model
Feature: Serving HTTPS with modern TLS only
  Protects clients and operators by refusing outdated TLS protocol versions on HTTPS listeners.

  @id:security.tls-certificates.001 @rule:security.tls.minimum-version-1-2 @inferred @source:config/model.go#Config
  Scenario Outline: HTTPS connections below TLS 1.2 are refused
    Given the proxy serves https://www.example.com with its built-in TLS settings
    When a client connects offering only <version>
    Then the connection is <outcome>

    Examples:
      | version | outcome  |
      | TLS 1.0 | refused  |
      | TLS 1.1 | refused  |
      | TLS 1.2 | accepted |
      | TLS 1.3 | accepted |
