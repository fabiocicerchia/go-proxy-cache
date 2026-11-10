@domain:configuration @area:config-loading @from:config/model
Feature: Applying built-in defaults and rejecting invalid settings
  Lets operators run the proxy with sensible behaviour when they leave settings out, and stops it from starting with settings that cannot be applied.

  Background:
    Given the operator starts the proxy without a config file entry or environment variable for the setting in question

  @id:configuration.config-loading.006 @rule:configuration.defaults.built-in-values @inferred @source:config/model.go#Config
  Scenario Outline: Unset settings fall back to the built-in defaults
    Given the operator has not set <setting>
    When the operator starts the proxy
    Then the effective <setting> is <default>

    Examples:
      | setting                     | default         |
      | HTTP port                   | 80              |
      | HTTPS port                  | 443             |
      | cacheable response statuses | 200, 301, 302   |
      | cacheable request methods   | GET, HEAD       |
      | balancing algorithm         | round-robin     |
      | internal listener           | 127.0.0.1:52021 |
      | request handling timeout    | 5s              |
      | shutdown grace period       | 5s              |
      | gzip compression            | off             |
      | collapsed forwarding        | off             |

  @id:configuration.config-loading.007 @rule:proxying.redirects.default-status-moved-permanently @inferred @source:config/model.go#Upstream @questionable
  Scenario: HTTP-to-HTTPS redirects use 301 unless the operator picks another status
    Given the operator has turned on redirecting HTTP to HTTPS
    And the operator has not set a redirect status code
    When a client requests http://www.example.com/page
    Then the response status is 301
    And the Location header is https://www.example.com/page
