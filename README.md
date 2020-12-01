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

[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/4469/badge)](https://bestpractices.coreinfrastructure.org/projects/4469)
[![Go Report Card](https://goreportcard.com/badge/github.com/fabiocicerchia/go-proxy-cache)](https://goreportcard.com/report/github.com/fabiocicerchia/go-proxy-cache)
[![codecov](https://codecov.io/gh/fabiocicerchia/go-proxy-cache/branch/main/graph/badge.svg)](https://codecov.io/gh/fabiocicerchia/go-proxy-cache)
[![Maintainability](https://api.codeclimate.com/v1/badges/6cf8c9ea02b75fccf8b5/maintainability)](https://codeclimate.com/github/fabiocicerchia/go-proxy-cache/maintainability)
[![Total alerts](https://img.shields.io/lgtm/alerts/g/fabiocicerchia/go-proxy-cache.svg?logo=lgtm&logoWidth=18)](https://lgtm.com/projects/g/fabiocicerchia/go-proxy-cache/alerts/)
![Builds](https://github.com/fabiocicerchia/go-proxy-cache/workflows/Builds/badge.svg)

</center>

## Features

### Small, Pragmatic and Easy to Use

- **Dockerized**
- **Compiled**
- **Easily Configurable**, via YAML or Environment Variables.
- **Self-Contained**, does not require Go, Git or any other software installed. Just run the binary or the container.

### Caching

- **Full Page Caching**, via Redis.
- **Cache Invalidation**, by calling HTTP Method `PURGE` on the resource URI.
- **Support Chunking**, by replicating exactly the same original amount.
- **Selective HTTP Status Codes/Methods**, allows caching for different response codes or HTTP methods.

### Load Balancing

- **HTTP & HTTPS Forward Traffic**
- **Load Balancing**, uses a list of IPs/Hostnames as load balanced backend servers (supported only Round-Robin).

### Security

- **HTTP/2 Support**
- **SSL/TLS Certificates via ACME**, provides automatic generation of SSL/TLS certificates from [Let's Encrypt](https://letsencrypt.org/) and any other ACME-based CA.
- **Using your own SSL/TLS Certificates**, optional.

### Reliability

- **Healthcheck Endpoint**, exposes the route `/healthcheck`.
- **Respecting HTTP Cache Headers**, `Vary`, `Cache-Control` and `Expires`.
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

## Configuration
## YAML

This is a simple (and not comprehensive) configuration:
```yaml
server:
  port:
    http: "80"
    https: "443"
  tls:
    certfile: server.pem
    keyfile: server.key
  forwarding:
    host: ~
    port: 443
    scheme: https
    endpoints:
      - 127.0.0.1
    http2https: true
    redirectstatuscode: 301

cache:
  host: localhost

domains:
  example_com:
    server:
      forwarding:
        host: example.com

  example_org:
    server:
      forwarding:
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
  -verbose
        enable verbose
  -version
        display version
[...]
```

For examples check the relative documentation in [docs/EXAMPLES.md](https://github.com/fabiocicerchia/go-proxy-cache/blob/main/docs/EXAMPLES.md)

## Common Errors

- `acme/autocert: server name component count invalid`  
  Let's Encrypt cannot be used locally, as described in [this thread](https://community.letsencrypt.org/t/can-i-test-lets-encrypt-client-on-localhost/15627)
- `acme/autocert: missing certificate`  
  Let's Encrypt cannot be used locally, as described in [this thread](https://community.letsencrypt.org/t/can-i-test-lets-encrypt-client-on-localhost/15627)
- `501 Not Implemented`  
  If there's no domain defined in the main configuration nor in the domain overrides, and a client will request an
  unknown domain the status `501` is returned.

## References

- [Proxy servers and tunneling](https://developer.mozilla.org/en-US/docs/Web/HTTP/Proxy_servers_and_tunneling)
- [Make resilient Go net/http servers using timeouts, deadlines and context cancellation](https://ieftimov.com/post/make-resilient-golang-net-http-servers-using-timeouts-deadlines-context-cancellation/)
- [So you want to expose Go on the Internet](https://blog.cloudflare.com/exposing-go-on-the-internet/)
- [Writing a very fast cache service with millions of entries in Go](https://allegro.tech/2016/03/writing-fast-cache-service-in-go.html)
- [RFC7234 - Hypertext Transfer Protocol (HTTP/1.1): Caching](https://tools.ietf.org/html/rfc7234#section-4.2.1)
- [The complete guide to Go net/http timeouts](https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/)
- [What Happens in a TLS Handshake? | SSL Handshake](https://www.cloudflare.com/en-gb/learning/ssl/what-happens-in-a-tls-handshake/)
- [A step by step guide to mTLS in Go](https://venilnoronha.io/a-step-by-step-guide-to-mtls-in-go)

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
