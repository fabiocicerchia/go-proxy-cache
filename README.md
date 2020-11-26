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
- Cache Circuit Breaker, bypassing Redis when not available
- GZip Compression (optional)
- Multiple domains
- Configure log verbosity level
- Configure error handler (stdout or file)

## Configuration

For more details about the server configuration check the relative documentation in [docs/CONFIGURATION.md](https://github.com/fabiocicerchia/go-proxy-cache/blob/main/docs/CONFIGURATION.md)

## Examples

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
