# Go Proxy Cache

<center>

![Logo](logo_small.png)

Simple Reverse Proxy with Caching, written in Go, backed by Redis.

[![MIT License](https://img.shields.io/badge/License-MIT-lightgrey.svg?longCache=true)](LICENSE)
[![Pull Requests](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?longCache=true)](https://github.com/fabiocicerchia/go-proxy-cache/pulls)
![Last Commit](https://img.shields.io/github/last-commit/fabiocicerchia/go-proxy-cache)
![Release Date](https://img.shields.io/github/release-date/fabiocicerchia/go-proxy-cache)

![Docker pulls](https://img.shields.io/docker/pulls/fabiocicerchia/go-proxy-cache.svg "Docker pulls")
![Docker stars](https://img.shields.io/docker/stars/fabiocicerchia/go-proxy-cache.svg "Docker stars")

[![Go Report Card](https://goreportcard.com/badge/github.com/fabiocicerchia/go-proxy-cache)](https://goreportcard.com/report/github.com/fabiocicerchia/go-proxy-cache)
[![codecov](https://codecov.io/gh/fabiocicerchia/go-proxy-cache/branch/main/graph/badge.svg)](https://codecov.io/gh/fabiocicerchia/go-proxy-cache)
![Builds](https://github.com/fabiocicerchia/go-proxy-cache/workflows/Builds/badge.svg)
</center>

## Features

  - Full Page Caching (via Redis)
  - Load Balancing (only Round-Robin)
  - `PURGE` Method to invalidate
  - HTTP & HTTPS Forward Traffic
  - HTTP/2 Support
  - SSL/TLS Certificates via ACME
  - Using your own SSL/TLS Certificates (optional)
  - Small, Pragmatic and Easy to Use
  - Easily Configurable (via YAML or Environment Variables)
  - Healthcheck Endpoint (`/healthcheck`)
  - Cache respecting HTTP Headers `Vary`, `Cache-Control` and `Expires`
  - Self-Contained, does not require Go, Git or any other software installed. Just run the binary or the container.
  - Tested (Unit, Functional & Linted & 0 Race Conditions Detected)

## Docker

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

## Configuration

### Environment Variables

- `SERVER_HTTP_PORT` = 80
- `SERVER_HTTPS_PORT` = 443
- `DEFAULT_TTL` = 0
- `FORWARD_HOST`
- `FORWARD_SCHEME`
- `LB_ENDPOINT_LIST`
- `TLS_AUTO_CERT` = 0
- `TLS_EMAIL`
- `TLS_CERT_FILE`
- `TLS_KEY_FILE`
- `TIMEOUT_READ` = 5
- `TIMEOUT_WRITE` = 5
- `TIMEOUT_IDLE` = 30
- `TIMEOUT_READ_HEADER` = 2
- `TIMEOUT_HANDLER` = 5
- `REDIS_DB` = 0
- `REDIS_HOST`
- `REDIS_PASSWORD`
- `REDIS_PORT` = 6379
- `CACHE_ALLOWED_METHODS` = HEAD,GET
- `CACHE_ALLOWED_STATUSES` = 200,301,302

### YAML

```yaml
server:
  port:
    http: "80"
    https: "443"
  ttl: 0
  tls:
    auto: 0
    email: info@fabiocicerchia.it
    certfile: server.pem
    keyfile: server.key
  timeouts:
    read: 5
    write: 5
    idle: 30
    readheader: 2
    handler: 5
  forwarding:
    host: fabiocicerchia.it
    scheme: https
    endpoints:
    - fabiocicerchia.it

cache:
  host: localhost
  port: "6379"
  password: ""
  db: 0
  allowedstatuses:
  - "200"
  - "301"
  - "302"
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

  - [Context timeouts and cancellation](https://ieftimov.com/post/make-resilient-golang-net-http-servers-using-timeouts-deadlines-context-cancellation/#context-timeouts-and-cancellation)
  - SSL Passthrough
  - WebSockets
  - GZip Compression
  - HTTP to HTTPS Redirects
  - Support Chunking
  - Cache [Circuit Breaker](https://github.com/sony/gobreaker)
  - Serve STALE cache
  - Cache Circuit Breaker
  - Cache Backends: Redis, [BigCache](https://github.com/allegro/bigcache), [FreeCache](https://github.com/coocood/freecache)
  - LB Algorithms
  - Tags
  - Improve Logging
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
