@domain:proxying @area:response-delivery @from:server/transport
Feature: Replaying stored responses to clients
  Visitors of a proxied site get a response from storage that looks like the origin's own: same status, headers and body, with connection-level headers removed and preload hints pushed over HTTP/2.

  Background:
    Given Given www.example.com is a domain served by the proxy
    And And the proxy holds a stored response for GET https://www.example.com/page

  @id:proxying.response-delivery.003 @rule:proxying.response-delivery.stored-response-keeps-status-headers-body @verified @source:server/transport/http.go#ServeCachedResponse @test:server/transport/http_test.go#TestServeCachedResponseSendsCachedStatusCode
  Scenario Outline: A stored response is replayed with its original status, headers and body
    Given Given the stored response has status <status>, header <header> set to "<value>" and body "<body>"
    When When a visitor requests https://www.example.com/page
    Then Then the visitor receives status <status>
    And And the response header <header> is "<value>"
    And And the response body is "<body>"

    Examples:
      | status | header       | value                     | body      |
      | 200    | Content-Type | text/plain                | ok        |
      | 301    | Location     | https://example.com/new   | moved     |
      | 302    | Location     | https://example.com/tmp   | found     |
      | 404    | Content-Type | text/html                 | not found |
      | 500    | Content-Type | text/plain                | boom      |
      | 301    | Location     | https://example.com/empty |           |

  @id:proxying.response-delivery.004 @rule:proxying.response-delivery.stored-response-drops-hop-by-hop-headers @inferred @source:server/transport/http.go#handleHeaders
  Scenario Outline: Connection-level headers from the origin are not replayed to visitors
    Given Given the stored response carries the header "<header>: <value>"
    When When a visitor requests https://www.example.com/page
    Then Then the response the visitor receives has no <dropped> header

    Examples:
      | header             | value                | dropped            |
      | Connection         | close                | Connection         |
      | Keep-Alive         | timeout=5            | Keep-Alive         |
      | Proxy-Authenticate | Basic realm="origin" | Proxy-Authenticate |
      | Proxy-Connection   | keep-alive           | Proxy-Connection   |
      | Transfer-Encoding  | chunked              | Transfer-Encoding  |
      | Upgrade            | h2c                  | Upgrade            |
      | Te                 | trailers             | Te                 |
      | Trailer            | Expires              | Trailer            |
      | Connection         | X-Origin-Debug       | X-Origin-Debug     |

  @id:proxying.response-delivery.005 @rule:proxying.response-delivery.stored-cache-status-not-replayed @inferred @source:server/transport/http.go#handleHeaders
  Scenario Outline: A replayed response reports today's cache status, not the one it was stored with
    Given Given the stored response carries the header "X-Go-Proxy-Cache-Status: MISS"
    And And the stored response is <freshness>
    When When a visitor requests https://www.example.com/page
    Then Then the response header X-Go-Proxy-Cache-Status has exactly one value, "<status>"

    Examples:
      | freshness | status |
      | fresh     | HIT    |
      | stale     | STALE  |

  @id:proxying.response-delivery.006 @rule:proxying.response-delivery.http2-push-preload-links @inferred @source:server/transport/http2.go#PushProxiedResources @questionable
  Scenario Outline: Over HTTP/2, resources the origin marks for preload are pushed with the stored page
    Given Given the stored response carries the header "Link: <link>"
    When When a visitor requests https://www.example.com/page over HTTP/2
    Then Then <pushed> is pushed to the visitor: <is_pushed>
    And And the response the visitor receives still carries the Link header

    Examples:
      | link                                        | pushed     | is_pushed |
      | </style.css>; as=style; rel=preload         | /style.css | yes       |
      | </nginx.png>; as=image; rel=preload; nopush | /nginx.png | no        |
      | </next.html>; rel=prefetch                  | /next.html | no        |
