@domain:proxying @area:response-delivery @from:server/response
Feature: Delivering proxied responses
  Ensures clients of a hosted domain receive the upstream's answer intact and in the encoding they were promised.

  Background:
    Given the proxy serves the domain www.example.com from an upstream

  @id:proxying.response-delivery.002 @rule:proxying.delivery.gzip-label-matches-body @inferred @source:server/handler/gzip.go#WrapResponseForGZip @questionable
  Scenario: A gzip-enabled domain labels responses as gzip but sends them uncompressed
    Given gzip is enabled for www.example.com
    And the upstream answers GET /app.js with status 200 and an uncompressed JavaScript body and no Content-Encoding
    When a client requests https://www.example.com/app.js with Accept-Encoding gzip
    Then the response carries Content-Encoding gzip
    And the response body is the upstream's uncompressed JavaScript, byte for byte
