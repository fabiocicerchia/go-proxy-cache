@domain:caching @area:freshness @from:server/response
Feature: Reporting cache status to clients
  Tells operators and clients, per response, whether the proxy served stored content and lets them force a fresh fetch.

  Background:
    Given Given the proxy serves the domain www.example.com from an upstream

  @id:caching.freshness.003 @rule:caching.status.reported-in-response-header @inferred @source:server/response/header.go#CacheStatusHeader
  Scenario Outline: Every proxied response reports whether it came from the proxy's store
    Given the page /news is <stored state>
    When a client requests https://www.example.com/news
    Then the response carries X-Go-Proxy-Cache-Status <status>
    And only one X-Go-Proxy-Cache-Status value is present, even if the stored copy was saved with one

    Examples:
      | stored state                  | status |
      | not stored by the proxy       | MISS   |
      | stored and still fresh        | HIT    |
      | stored but past its freshness | STALE  |
