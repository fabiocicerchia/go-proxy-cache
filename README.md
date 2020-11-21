# Go Proxy Cache

<center>

![Logo](https://github.com/fabiocicerchia/go-proxy-cache/raw/main/logo_small.png)

Simple Reverse Proxy with Caching, written in Go, using Redis.

[![MIT License](https://img.shields.io/badge/License-MIT-lightgrey.svg?longCache=true)](LICENSE)
[![Pull Requests](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?longCache=true)](https://github.com/fabiocicerchia/go-proxy-cache/pulls)
![Last Commit](https://img.shields.io/github/last-commit/fabiocicerchia/go-proxy-cache)
![Release Date](https://img.shields.io/github/release-date/fabiocicerchia/go-proxy-cache)

![Docker pulls](https://img.shields.io/docker/pulls/fabiocicerchia/go-proxy-cache.svg "Docker pulls")
![Docker stars](https://img.shields.io/docker/stars/fabiocicerchia/go-proxy-cache.svg "Docker stars")

[![Go Report Card](https://goreportcard.com/badge/github.com/fabiocicerchia/go-proxy-cache)](https://goreportcard.com/report/github.com/fabiocicerchia/go-proxy-cache)
[![codecov](https://codecov.io/gh/fabiocicerchia/go-proxy-cache/branch/main/graph/badge.svg)](https://codecov.io/gh/fabiocicerchia/go-proxy-cache)
[![Maintainability](https://api.codeclimate.com/v1/badges/6cf8c9ea02b75fccf8b5/maintainability)](https://codeclimate.com/github/fabiocicerchia/go-proxy-cache/maintainability)
[![Total alerts](https://img.shields.io/lgtm/alerts/g/fabiocicerchia/go-proxy-cache.svg?logo=lgtm&logoWidth=18)](https://lgtm.com/projects/g/fabiocicerchia/go-proxy-cache/alerts/)
![Builds](https://github.com/fabiocicerchia/go-proxy-cache/workflows/Builds/badge.svg)

</center>

## Features

- Full Page Caching (via Redis)
- Load Balancing (only Round-Robin)
- `PURGE` Method to invalidate
- HTTP & HTTPS Forward Traffic
- HTTP/2 Support
- SSL/TLS Certificates via ACME
- HTTP to HTTPS Redirects
- Using your own SSL/TLS Certificates (optional)
- Small, Pragmatic and Easy to Use
- Easily Configurable (via YAML or Environment Variables)
- Healthcheck Endpoint (`/healthcheck`)
- Cache respecting HTTP Headers `Vary`, `Cache-Control` and `Expires`
- Self-Contained, does not require Go, Git or any other software installed. Just run the binary or the container.
- Tested (Unit, Functional & Linted & 0 Race Conditions Detected)
- Support Chunking (by replicating exactly the same original amount)

## Examples

### Docker

```console
$ docker run \
    -it --rm -n goproxycache \
    --env SERVER_HTTP_PORT=80 \
    --env SERVER_HTTPS_PORT=443 \
    --env DEFAULT_TTL=0 \
    --env FORWARD_HOST=www.google.com \
    --env FORWARD_SCHEME=https \
    --env LB_ENDPOINT_LIST=www.google.com \
    --env REDIS_HOST=localhost \
    --env REDIS_PORT=6379 \
    --env REDIS_PASSWORD= \
    --env REDIS_DB=0 \
    -p 8080:80
    -p 8443:443
    fabiocicerchia/go-proxy-cache
```

### PURGE

```concole
$ curl -vX PURGE http://localhost/cached/page
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to localhost (127.0.0.1) port 80 (#0)
> PURGE / HTTP/1.1
> Host: localhost
> User-Agent: curl/7.64.1
> Accept: */*
>
< HTTP/1.1 200 OK
< Date: Thu, 19 Nov 2020 11:21:45 GMT
< Content-Length: 2
< Content-Type: text/plain; charset=utf-8
<
* Connection #0 to host localhost left intact
OK* Closing connection 0
```

```concole
$ curl -vX PURGE http://localhost/page/not/cached
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to localhost (127.0.0.1) port 80 (#0)
> PURGE / HTTP/1.1
> Host: localhost
> User-Agent: curl/7.64.1
> Accept: */*
>
< HTTP/1.1 404 Not Found
< Date: Thu, 19 Nov 2020 11:23:36 GMT
< Content-Length: 2
< Content-Type: text/plain; charset=utf-8
<
* Connection #0 to host localhost left intact
KO* Closing connection 0
```

### HTTP/2

```console
$ curl -4 -s -I -w '%{http_version}\n' -o /dev/null http://localhost
1.1
$ curl -4 -k -s -I -w '%{http_version}\n' -o /dev/null https://localhost
2
```

### HealthCheck

```console
$ curl -v http://localhost/healthcheck
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to localhost (127.0.0.1) port 80 (#0)
> GET /healthcheck HTTP/1.1
> Host: localhost
> User-Agent: curl/7.64.1
> Accept: */*
>
< HTTP/1.1 200 OK
< Date: Thu, 19 Nov 2020 11:26:37 GMT
< Content-Length: 17
< Content-Type: text/plain; charset=utf-8
<
HTTP OK
REDIS OK
* Connection #0 to host localhost left intact
* Closing connection 0
```

## Configuration

> ![Timeouts](https://blog.cloudflare.com/content/images/2016/06/Timeouts-001.png)
> - [The complete guide to Go net/http timeouts](https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/)

### Environment Variables

- `SERVER_HTTP_PORT` = 80
- `SERVER_HTTPS_PORT` = 443
- `DEFAULT_TTL` = 0
- `FORWARD_HOST`
- `FORWARD_SCHEME`
- `LB_ENDPOINT_LIST`
- `HTTP2HTTPS` = 0
- `REDIRECT_STATUS_CODE` = 301
- `TLS_AUTO_CERT` = 0
- `TLS_EMAIL`
- `TLS_CERT_FILE`
- `TLS_KEY_FILE`
- `TIMEOUT_READ` = 50000000000
- `TIMEOUT_READ_HEADER` = 20000000000
- `TIMEOUT_WRITE` = 50000000000
- `TIMEOUT_IDLE` = 300000000000
- `TIMEOUT_HANDLER` = 50000000000
- `REDIS_DB` = 0
- `REDIS_HOST`
- `REDIS_PASSWORD`
- `REDIS_PORT` = 6379
- `CACHE_ALLOWED_METHODS` = HEAD,GET
- `CACHE_ALLOWED_STATUSES` = 200,301,302

### YAML

```yaml
server:
  # --- GENERIC
  port:
    http: "80"
    https: "443"
  # --- TLS
  tls:
    # Automatic Certificate Management Environment
    # Provides automatic generation of SSL/TLS certificates from Let's Encrypt
    # and any other ACME-based CA.
    # Default: false (need to provide `certfile` and `keyfile`)
    auto: false
    # Email optionally specifies a contact email address.
    # This is used by CAs, such as Let's Encrypt, to notify about problems with
    # issued certificates.
    email: info@fabiocicerchia.it
    # Pair or files: the certificate and the key.
    # Used by LoadX509KeyPair to read and parse a public/private key pair from a
    # pair of files. The files must contain PEM encoded data. The certificate
    # file may contain intermediate certificates following the leaf certificate
    # to form a certificate chain.
    certfile: server.pem
    keyfile: server.key
  # --- TIMEOUT
  timeout:
    # It is the maximum duration for reading the entire request, including the
    # body.
    # Because it does not let Handlers make per-request decisions on each
    # request body's acceptable deadline or upload rate, most users will prefer
    # to use `readheader`. It is valid to use them both.
    read: 5000000000
    # It is the amount of time allowed to read request headers. The connection's
    # read deadline is reset after reading the headers and the Handler can
    # decide what is considered too slow for the body. If it is zero, the value
    # of `read` is used. If both are zero, there is no timeout.
    readheader: 2000000000
    # It is the maximum duration before timing out writes of the response. It is
    # reset whenever a new request's header is read. Like `read`, it does not
    # let Handlers make decisions on a per-request basis.
    write: 5000000000
    # It is the maximum amount of time to wait for the next request when
    # keep-alives are enabled. If is zero, the value of `read` is used. If both
    # ara zero, there is no timeout.
    idle: 20000000000
    handler: 5000000000
  # --- FORWARDING
  forwarding:
    # Hostname to be used for requests forwarding.
    host: fabiocicerchia.it
    # Endpoint scheme to be used when forwarding traffic.
    # Default: incoming connection.
    # Values: http, https.
    scheme: https
    # List of IPs/Hostnames to be used as load balanced backend servers.
    # They'll be selected using a round robin algorithm.
    endpoints:
    - fabiocicerchia.it
    # Forces redirect from HTTP to HTTPS.
    # Default: false
    http2https: false

# --- TIMEOUT
cache:
  # --- REDIS SERVER
  host: localhost
  port: "6379"
  password: ""
  db: 0
  # --- TTL
  # Fallback storage TTL when saving the cache when no header is specified.
  # It follows the order:
  #  - If the cache is shared and the s-maxage response directives present, use
  #    its value, or
  #  - If the max-age response directive is present, use its value, or
  #  - If the Expires response header field is present, use its value minus the
  #    value of the Date response header field, or
  #  - Otherwise, no explicit expiration time is present in the response.
  #    A heuristic freshness lifetime might be applicable.
  # Default: 0
  ttl: 0
  # --- ALLOWED VALUES
  # Allows caching for different response codes.
  # Default: 200, 301, 302
  allowedstatuses:
  - "200"
  - "301"
  - "302"
  # If the client request method is listed in this directive then the response
  # will be cached. "GET" and "HEAD" methods are always added to the list,
  # though it is recommended to specify them explicitly.
  # Default: HEAD, GET
  allowedmethods:
  - HEAD
  - GET

```

## Common Errors

- `acme/autocert: server name component count invalid`  
  Let's Encrypt cannot be used locally, as described in [this thread](https://community.letsencrypt.org/t/can-i-test-lets-encrypt-client-on-localhost/15627)
- `acme/autocert: missing certificate`  
  Let's Encrypt cannot be used locally, as described in [this thread](https://community.letsencrypt.org/t/can-i-test-lets-encrypt-client-on-localhost/15627)

## References

- [Proxy servers and tunneling](https://developer.mozilla.org/en-US/docs/Web/HTTP/Proxy_servers_and_tunneling)
- [Make resilient Go net/http servers using timeouts, deadlines and context cancellation](https://ieftimov.com/post/make-resilient-golang-net-http-servers-using-timeouts-deadlines-context-cancellation/)
- [So you want to expose Go on the Internet](https://blog.cloudflare.com/exposing-go-on-the-internet/)
- [Writing a very fast cache service with millions of entries in Go](https://allegro.tech/2016/03/writing-fast-cache-service-in-go.html)
- [RFC7234 - Hypertext Transfer Protocol (HTTP/1.1): Caching](https://tools.ietf.org/html/rfc7234#section-4.2.1)

## TODO

- Support Chunking
- https://stackoverflow.com/questions/26769626/send-a-chunked-http-response-from-a-go-server 30m
- Cache [Circuit Breaker](https://github.com/sony/gobreaker) 30m
- Test server timeout with custom handlers 15m
- [Context timeouts and cancellation](https://ieftimov.com/post/make-resilient-golang-net-http-servers-using-timeouts-deadlines-context-cancellation/#context-timeouts-and-cancellation)
- [Check Timeouts](https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/)
- [GZip Compression](github.com/NYTimes/gziphandler) + Config Flag
- [SSL Passthrough](https://stackoverflow.com/a/35399699/888162)
- [Go Language - Web Application Secure Coding Practices](https://github.com/OWASP/Go-SCP/raw/master/dist/go-webapp-scp.pdf)
- [HTTP/2 Adventure in the Go World](https://posener.github.io/http2/)
- https://cipherli.st/
- Check [SSL Labs Score](https://blog.bracelab.com/achieving-perfect-ssl-labs-score-with-go)
- Use [X-Forwarded-Proto](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Forwarded-Proto)
- Configure log verbosity level
- [tlsfuzzer](https://github.com/tlsfuzzer/tlsfuzzer)
- [Building Web Servers in Go](https://getgophish.com/blog/post/2018-12-02-building-web-servers-in-go/)
- [TLS mutual authentication with golang and nginx](https://medium.com/rahasak/tls-mutual-authentication-with-golang-and-nginx-937f0da22a0e)
- Check file descriptors usage
- LB Algorithms
- AB Benchmarks
- WebSockets
- SOCKS4/SOCKS5
- Serve STALE cache
- Cache Backends: Redis, [BigCache](https://github.com/allegro/bigcache), [FreeCache](https://github.com/coocood/freecache)
- Tags
- Define eviction: LRU, LFU, ...
- Byte-Range Cache
- Dashboard
- CLI Monitor
- HTTP/3

## License

MIT License

Copyright (c) 2020 Fabio Cicerchia <info@fabiocicerchia.it>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
