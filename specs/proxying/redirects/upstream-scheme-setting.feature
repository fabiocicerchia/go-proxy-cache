@domain:proxying @area:redirects @from:utils/scheme
Feature: Choosing the upstream scheme
  Lets operators set which scheme (http or https) the proxy uses to reach the upstream, so traffic reaches backends over the protocol they expect.

  Background:
    Given the operator runs the proxy with a YAML config file
    And the global upstream is "backend.example.com" with endpoint "10.0.0.10"

  @id:proxying.redirects.002 @rule:proxying.upstream-scheme.case-insensitive @verified @source:utils/scheme/scheme.go#NormalizeScheme @test:utils/scheme/scheme_test.go#TestNormalizeHTTP @test:utils/scheme/scheme_test.go#TestNormalizeHTTPS
  Scenario Outline: The upstream scheme is accepted in any letter case
    Given the config file sets the global upstream scheme to "<configured>"
    When a client sends a request over <incoming> to backend.example.com
    Then the proxy forwards the request to the upstream over "<forwarded>"

    Examples:
      | configured | incoming | forwarded |
      | https      | http     | https     |
      | HTTPS      | http     | https     |
      | HtTpS      | http     | https     |
      | HTTP       | https    | http      |
      | HttP       | https    | http      |

  @id:proxying.redirects.003 @rule:proxying.upstream-scheme.unsupported-falls-back-to-incoming @verified @source:utils/scheme/scheme.go#NormalizeScheme @test:utils/scheme/scheme_test.go#TestNormalizeNonExisting @questionable
  Scenario Outline: An unsupported upstream scheme silently falls back to the client's scheme
    Given the config file sets the global upstream scheme to "<configured>"
    And no upstream scheme is set in the environment
    When a client sends a request over <incoming> to backend.example.com
    Then the proxy starts without any warning about the scheme
    And the proxy forwards the request to the upstream over "<incoming>"

    Examples:
      | configured | incoming |
      | wss        | https    |
      | ws         | http     |
      | ftp        | http     |
      |            | https    |
