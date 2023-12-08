# Go Proxy Cache

<center>

![Logo](https://github.com/fabiocicerchia/go-proxy-cache/raw/main/docs/assets/logo_small.png)

Simple Reverse Proxy with Caching, written in Go, using Redis.  
    >>> **(semi) production-ready** <<<

[![MIT License](https://img.shields.io/badge/License-MIT-brightgreen.svg?longCache=true)](LICENSE)
[![Pull Requests](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?longCache=true)](https://github.com/fabiocicerchia/go-proxy-cache/pulls)
![Maintenance](https://img.shields.io/maintenance/yes/2023)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)  
  
![Last Commit](https://img.shields.io/github/last-commit/fabiocicerchia/go-proxy-cache)
![Release Date](https://img.shields.io/github/release-date/fabiocicerchia/go-proxy-cache)
![GitHub all releases](https://img.shields.io/github/downloads/fabiocicerchia/go-proxy-cache/total)

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/fabiocicerchia/go-proxy-cache)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/fabiocicerchia/go-proxy-cache)

![Docker pulls](https://img.shields.io/docker/pulls/fabiocicerchia/go-proxy-cache "Docker pulls")
![Docker stars](https://img.shields.io/docker/stars/fabiocicerchia/go-proxy-cache "Docker stars")

![GitHub Workflow Status](https://img.shields.io/github/workflow/status/fabiocicerchia/go-proxy-cache/Builds)

[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/4469/badge)](https://bestpractices.coreinfrastructure.org/projects/4469)
[![BCH compliance](https://bettercodehub.com/edge/badge/fabiocicerchia/go-proxy-cache?branch=main)](https://bettercodehub.com/results/fabiocicerchia/go-proxy-cache)
[![Go Report Card](https://goreportcard.com/badge/github.com/fabiocicerchia/go-proxy-cache)](https://goreportcard.com/report/github.com/fabiocicerchia/go-proxy-cache)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Ffabiocicerchia%2Fgo-proxy-cache.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Ffabiocicerchia%2Fgo-proxy-cache?ref=badge_shield)  
[![codecov](https://codecov.io/gh/fabiocicerchia/go-proxy-cache/branch/main/graph/badge.svg)](https://codecov.io/gh/fabiocicerchia/go-proxy-cache)
[![Maintainability](https://img.shields.io/codeclimate/maintainability/fabiocicerchia/go-proxy-cache)](https://codeclimate.com/github/fabiocicerchia/go-proxy-cache/maintainability)
[![Technical Debt](https://img.shields.io/codeclimate/tech-debt/fabiocicerchia/go-proxy-cache)](https://codeclimate.com/github/fabiocicerchia/go-proxy-cache/maintainability)
[![Total alerts](https://img.shields.io/lgtm/alerts/g/fabiocicerchia/go-proxy-cache.svg)](https://lgtm.com/projects/g/fabiocicerchia/go-proxy-cache/alerts/)

</center>

---

## 💗 Support the Project 💗

This project is only maintained by one person, [Fabio Cicerchia](https://github.com/fabiocicerchia).  
It started as a simple caching service, now it has a lot of pro functionalities just for FREE 😎  
Maintaining a project is a very time consuming activity, especially when done alone 💪
I really want to make this project better and become super cool 🚀

Two commercial versions have been planned: [PRO and PREMIUM](https://kodebeat.com/goproxycache.html).  

The development of the COMMUNITY version will continue, but priority will be given to the [COMMERCIAL versions](https://kodebeat.com/goproxycache.html).  
- If you'd like to support this open-source project I'll appreciate any kind of [contribution](https://github.com/sponsors/fabiocicerchia).
- If you'd like to sponsor the commercial version, please [get in touch with me](mail:info@fabiocicerchia.it).

---

## How it works

When the request is cached:

```text
        .---------.       .---------.       .---------.
        |         |       |         |       |         |
        |         |       |         |       |         |
you --->|---->----|--->---|---->----|--->---|-->--.   |
        |         |       |         |       |     |   |
    <---|----<----|---<---|----<----|---<---|--<--'   |
        `---------´       `---------´       `---------´
          network        go-proxy-cache        redis
```

When the request is not cached:

```text
          website
            ,_,
            | |
        .---+-+---.       .---------.       .---------.
        |   | '-->|--->---|---->----|--->---|-->--,   |
        |   '-<---|---<---|<--,     |       |     |   |
        |         |       |   |     |       |     |   |
you --->|---->----|--->---|---'     |       |     |   |
        |         |       |         |       |     |   |
    <---|----<----|---<---|----<----|---<---|--<--'   |
        `---------´       `---------´       `---------´
          network        go-proxy-cache        redis
```

## Features

### Small, Pragmatic and Easy to Use

- **Dockerized**
- **Compiled**
- **Easily Configurable**, via YAML or Environment Variables.
- **Self-Contained**, does not require Go, Git or any other software installed. Just run the binary or the container.

### Caching

- **Full Page Caching**, via Redis.
- **Cache Invalidation**, by calling HTTP Method `PURGE` on the resource URI.
- **Cache Bypass**, by using the HTTP Header `X-Go-Proxy-Cache-Force-Fresh` the request will always be fresh.
- **Support Chunking**, by replicating exactly the same original amount.
- **Selective HTTP Status Codes/Methods**, allows caching for different response codes or HTTP methods.
- **ETag Support**, generating non-weak tags, handling `304 Not Modified`, managing HTTP headers `If-Modified-Since`, `If-Unmodified-Since`, `If-None-Match`, `If-Match`.  
  ETag wrapper doesn't work well with WebSocket and HTTP/2.
- **Cache Stampede Prevention**, delaying invalidation request to the backend using an extra small random TTL (between 5s and 10s).
- **Serving Stale Content**, used mainly for avoiding cache stampede, for maximum 10s.
- **Upstream DNS Resolution Cache**, the upstream hostname will be cached to speed up the response and avoid the DNS resolution at each request.

### Load Balancing

- **HTTP & HTTPS Forward Traffic**
- **Load Balancing**, uses a list of IPs/Hostnames as load balanced backend servers.
- **Multiple Algorithms Available**, choose among IP Hash, Least Connections, Random or Round-Robin.
- **Support for HTTP Basic Auth**, it's possible to provide the HTTP Basic Auth for each endpoint (by specify user:pass in the URL).

### Security

- **HTTP/2 Support**, HTTP/2 Pusher achievable only if upstream implements [HTTP header `Link`](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Link). Server Push is deprecated (since not really supported in the browsers).
- **SSL/TLS Certificates via ACME**, provides automatic generation of SSL/TLS certificates from [Let's Encrypt](https://letsencrypt.org/) and any other ACME-based CA.
- **Using your own SSL/TLS Certificates**, optional.

### Reliability

- **Healthcheck Endpoint**, exposes the route `/healthcheck` (internally).
- **Upstream Healthcheck**, verifies periodically if upstream nodes are healthy.
- **Respecting HTTP Cache Headers**, `Vary`, `ETag`, `Cache-Control` and `Expires`.
- **Fully Tested**, Unit, Functional & Linted & 0 Race Conditions Detected.
- **Cache Circuit Breaker**, bypassing Redis when not available.

### Scaling

- **Multiple domains**, override and fine-tune the global settings per domain.

### Customisations

- **HTTP to HTTPS Redirects**, optional, status code to be used when redirecting HTTP to HTTPS.
- **GZip Compression**, optional.
- **Server Timeouts**, it is possible to configure in details the server overall timeouts (read, write, headers, handler, idle).
- **Fine tuning circuit-breaker and TLS settings**, it is possible to adjust the settings about thresholds, timeouts and failure rate.
- **Configure error handler**, stdout or file.
- **Debug/Verbose mode**, it is possible to have additional levels of details by settings the flags `-verbose` or `-debug`.

### Logging

- **Request Tracing**, each line in logs has a RequestID to easily identify the response flow.
- **OpenTelemetry Tracing**, each request has a deep tracing with Jaeger (optional).
- **Prometheus Endpoint**, exposes the route `/metrics` (internally) to serve Prometheus metrics.
- **Support for Sentry & Syslog**, all warning/error logs can be forwarded to Sentry and/or Syslog.

## Configuration

## YAML

This is a simple (and not comprehensive) configuration:

```yaml
server:
  port:
    http: "80"
    https: "443"
  tls:
    cert_file: server.pem
    key_file: server.key
  upstream:
    host: ~
    port: 443
    scheme: https
    endpoints:
      - 127.0.0.1
    http_to_https: true
    redirect_status_code: 301

cache:
  hosts: 
    - localhost:6379

domains:
  example_com:
    server:
      upstream:
        host: example.com

  example_org:
    server:
      upstream:
        host: example.org
```

For more details about the full server configuration check the relative documentation in [docs/CONFIGURATION.md](https://github.com/fabiocicerchia/go-proxy-cache/blob/main/docs/CONFIGURATION.md)

## Examples

## CLI

```console
$ go-proxy-cache -h
Usage of go-proxy-cache:
  -config string
        config file (default "config.yml")
  -debug
        enable debug
  -log string
        log file (default stdout)
  -test
        test configuration
  -verbose
        enable verbose
  -version
        display version
[...]
```

For examples check the relative documentation in [docs/EXAMPLES.md](https://github.com/fabiocicerchia/go-proxy-cache/blob/main/docs/EXAMPLES.md)

## Release Cycle

- Bug-fixes (e.g. `1.1.1`, `1.1.2`, `1.2.1`, `1.2.3`) are released as needed (no additional features are delivered in those versions, bug-fixes only).
- Each version is supported until the next one is released (e.g. `1.1.x` will be supported until `1.2.0` is out).
- We use [Semantic Versioning](https://semver.org/).

## Common Errors

- `acme/autocert: server name component count invalid`  
  Let's Encrypt cannot be used locally, as described in [this thread](https://community.letsencrypt.org/t/can-i-test-lets-encrypt-client-on-localhost/15627)
- `acme/autocert: missing certificate`  
  Let's Encrypt cannot be used locally, as described in [this thread](https://community.letsencrypt.org/t/can-i-test-lets-encrypt-client-on-localhost/15627)
- `501 Not Implemented`  
  If there's no domain defined in the main configuration nor in the domain overrides, and a client will request an
  unknown domain the status `501` is returned.
- WebSocket and TimeoutHandler are not working together, because TimeoutHandler doesn't support Hijacker, so in order to have WebSocket support the setting `TimeoutHandler` must be set to `-1`.
- `context deadline exceeded`  
  The reason is because the timeout on the context.Context of the client side of the request is shorter than the timeout
  in the server side handler. This means that the client gives up before any response is written.

## References

- [Proxy servers and tunneling](https://developer.mozilla.org/en-US/docs/Web/HTTP/Proxy_servers_and_tunneling)
- [Make resilient Go net/http servers using timeouts, deadlines and context cancellation](https://ieftimov.com/post/make-resilient-golang-net-http-servers-using-timeouts-deadlines-context-cancellation/)
- [So you want to expose Go on the Internet](https://blog.cloudflare.com/exposing-go-on-the-internet/)
- [Writing a very fast cache service with millions of entries in Go](https://allegro.tech/2016/03/writing-fast-cache-service-in-go.html)
- [RFC7234 - Hypertext Transfer Protocol (HTTP/1.1): Caching](https://tools.ietf.org/html/rfc7234#section-4.2.1)
- [The complete guide to Go net/http timeouts](https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/)
- [What Happens in a TLS Handshake? | SSL Handshake](https://www.cloudflare.com/en-gb/learning/ssl/what-happens-in-a-tls-handshake/)
- [A step by step guide to mTLS in Go](https://venilnoronha.io/a-step-by-step-guide-to-mtls-in-go)
- [Learning HTTP caching in Go](https://www.sanarias.com/blog/115LearningHTTPcachinginGo)
- [Nginx HTTP2 Server Push](https://ops.tips/blog/nginx-http2-server-push/)
- [Introducing HTTP/2 Server Push with NGINX 1.13.9](https://www.nginx.com/blog/nginx-1-13-9-http2-server-push)
- [Preload - W3C Editor's Draft 20 August 2020](https://w3c.github.io/preload/#server-push)
- [Web Linking](https://tools.ietf.org/html/rfc5988)
- [HTTP Health Checks](https://docs.nginx.com/nginx/admin-guide/load-balancer/http-health-check/)
- [Types of load balancing algorithms](https://www.cloudflare.com/en-gb/learning/performance/types-of-load-balancing-algorithms/)

## License

## OpenSSL

This product includes software developed by the OpenSSL Project for use in the
OpenSSL Toolkit. ([http://www.openssl.org/](http://www.openssl.org/))

## Go Proxy Cache

MIT License

Copyright (c) 2023 Fabio Cicerchia <info@fabiocicerchia.it>

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

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Ffabiocicerchia%2Fgo-proxy-cache.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Ffabiocicerchia%2Fgo-proxy-cache?ref=badge_large)
