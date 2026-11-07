@domain:proxying @area:redirects @from:server/handler
Feature: Redirecting plain HTTP to HTTPS
  Lets operators force visitors onto HTTPS with the redirect status they choose.

  Background:
    Given the domain "shop.example.com" has HTTP-to-HTTPS redirection enabled

  @id:proxying.redirects.001 @rule:proxying.redirects.http-to-https @verified @source:server/handler/redirect.go#RedirectToHTTPS @test:server/handler/http_functional_test.go#TestHTTPEndToEndCallRedirect @test:server/handler/http_functional_test.go#TestHTTPEndToEndCallWithHTTPSRedirect @questionable
  Scenario Outline: Plain HTTP visitors are sent to the same address over HTTPS
    Given the domain's redirect status code is <status>
    When a client sends GET /cart?id=7 over plain HTTP
    Then the response status is <status>
    And the Location header is "https://shop.example.com/cart?id=7"
    And nothing is forwarded to the upstream

    Examples:
      | status |
      | 301    |
      | 302    |
