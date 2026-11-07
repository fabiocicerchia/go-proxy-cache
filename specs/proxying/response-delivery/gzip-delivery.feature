@domain:proxying @area:response-delivery @from:server/handler
Feature: Compressed delivery
  Lets operators serve gzip-encoded responses to clients that accept them.

  @id:proxying.response-delivery.001 @rule:proxying.delivery.gzip-labelled-for-accepting-clients @verified @source:server/handler/gzip.go#WrapResponseForGZip @test:server/handler/gzip_test.go#TestWrapResponseForGZipWhenAcceptEncodingGZip @test:server/handler/gzip_test.go#TestWrapResponseForGZipWhenNoAcceptEncoding @questionable
  Scenario Outline: Clients that accept gzip get responses labelled gzip on domains with gzip enabled
    Given gzip is enabled for shop.example.com
    And the upstream answers /cart with an uncompressed body
    When a client sends GET /cart with Accept-Encoding "<accept>"
    Then the response Content-Encoding is <encoding>
    And the body is the upstream's bytes, unchanged

    Examples:
      | accept        | encoding |
      | gzip, deflate | "gzip"   |
      | identity      | absent   |
