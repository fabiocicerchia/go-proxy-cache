# Tracing

Go Proxy Cache supports OpenTelemetry, specifically Jaeger.

Use the environment variable `TRACING_ENV` to customise the tracing.

There is a unique key used in order to be able to match the request against multiple services.
The key in OpenTelemetry is `request.id`, it is also sent to the upstream backend as an additional HTTP header (ie. `X-Go-Proxy-Cache-Request-ID`).

For every request these are the tracing spans (and relative tags) created:

- `server.handle_request`
  - `purge.status` - Status of PURGE on resource
  - `request.full_url` - Full Request URL
  - `request.host` - Request's Host
  - `request.id` - Request Unique ID
  - `request.is_legit.conf_hostname` - Configuration Host value
  - `request.is_legit.conf_port` - Configuration port value
  - `request.is_legit.hostname_matches` - Request is legit as request's host matches the configuration
  - `request.is_legit.port_matches` - Request is legit as request's port matches the configuration
  - `request.is_legit.req_hostname` - Request's Host value
  - `request.is_legit.req_port` - Request's port value
  - `request.method` - Request's HTTP Request Method
  - `request.scheme` - Request's URL Scheme
  - `request.url` - Request's URL
  - `request.websocket` - Is Request using a WebSocket?
  - `response.location` - HTTP 301 HTTP Location
  - `response.must_serve_original_response.etag_already_present` - Should serve original response if an ETag is already set in upstream
  - `response.must_serve_original_response.etag_present` - Upstream ETag HTTP Header value
  - `response.must_serve_original_response.no_buffered_content` - Should serve original response if there is no content
  - `response.must_serve_original_response.no_hash_computed` - Should serve original response if no ETag has been generated
  - `response.must_serve_original_response.response_204` - Should serve original response if response is 204 no content
  - `response.must_serve_original_response.response_not_2xx` - Should serve original response if response is not successful
  - `response.must_serve_original_response.response_status_code` - Upstream HTTP Status Code
  - `response.status_code` - Response Status Code
- `server.handle_healthcheck`
  - `response.status_code` - Response Status Code
- `handler.handle_http_request_and_proxy`
  - `cache.cacheable` - Is Resource Cacheable?
  - `cache.cached` - Was the Response Cached?
  - `cache.forced_fresh` - Bypass Cached Content (requested by user)
  - `cache.stale` - Was the Cached Response Stale?
- `handler.serve_cached_content`
  - `cache.stale` - Was the Cached Response Stale?
  - `response.status_code` - Response Status Code
- `handler.serve_reverse_proxy_http`
  - `cache.cacheable` - Is Resource Cacheable?
  - `cache.cached` - Was the Response Cached?
  - `cache.forced_fresh` - Bypass Cached Content (requested by user)
  - `cache.stale` - Was the Cached Response Stale?
  - `proxy.endpoint` - Upstream URL
- `handler.store_response`
  - `storage.cached` - Has the Response been saved?
- `handler.serve_reverse_proxy_ws`
  - `cache.cacheable` - Is Resource Cacheable?
  - `cache.cached` - Was the Response Cached?
  - `cache.forced_fresh` - Bypass Cached Content (requested by user)
  - `cache.stale` - Was the Cached Response Stale?
  - `proxy.endpoint` - Upstream URL
