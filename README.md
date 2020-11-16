# Go Proxy Cache

[![MIT License](https://img.shields.io/badge/License-MIT-lightgrey.svg?longCache=true)](LICENSE)
[![Pull Requests](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?longCache=true)](https://github.com/fabiocicerchia/go-proxy-cache/pulls)
![Last Commit](https://img.shields.io/github/last-commit/fabiocicerchia/go-proxy-cache)
![Release Date](https://img.shields.io/github/release-date/fabiocicerchia/go-proxy-cache)

![Docker pulls](https://img.shields.io/docker/pulls/fabiocicerchia/go-proxy-cache.svg "Docker pulls")
![Docker stars](https://img.shields.io/docker/stars/fabiocicerchia/go-proxy-cache.svg "Docker stars")

[![Go Report Card](https://goreportcard.com/badge/github.com/fabiocicerchia/go-proxy-cache)](https://goreportcard.com/report/github.com/fabiocicerchia/go-proxy-cache)
[![codecov](https://codecov.io/gh/fabiocicerchia/go-proxy-cache/branch/main/graph/badge.svg)](https://codecov.io/gh/fabiocicerchia/go-proxy-cache)
![Builds](https://github.com/fabiocicerchia/go-proxy-cache/workflows/Builds/badge.svg)

Simple caching proxy written in golang backed by redis.

## Features

  - HTTP Forward Traffic
  - Full Page Caching (via Redis)
  - Load Balancing (only Round-Robin)
  - Small, Pragmatic and Easy to Use
  - Easily Configurable (via YAML or Environment Variables)
  - Cache respecting HTTP Headers `Vary`, `Cache-Control` and `Expires`
  - Self-Contained, does not require Go, Git or any other software installed. Just run the binary or the container.
  - Tested (Unit, Functional & Linted & 0 Race Conditions Detected)

## Docker

```console
$ docker run \
    -it --rm -n goproxycache \
    --env SERVER_PORT=8080 \
    --env DEFAULT_TTL=0 \
    --env FORWARD_HOST=www.google.com \
    --env FORWARD_SCHEME=https \
    --env LB_ENDPOINT_LIST=www.google.com \
    --env REDIS_HOST=localhost \
    --env REDIS_PORT=6379 \
    --env REDIS_PASSWORD= \
    --env REDIS_DB=0 \
    fabiocicerchia/go-proxy-cache
```

## Configuration

### Environment Variables

- `SERVER_PORT` = 8080
- `DEFAULT_TTL` = 0
- `FORWARD_HOST`
- `FORWARD_SCHEME`
- `LB_ENDPOINT_LIST`
- `REDIS_DB` = 0
- `REDIS_HOST`
- `REDIS_PASSWORD`
- `REDIS_PORT` = 6379
- `CACHE_ALLOWED_METHODS` = HEAD,GET
- `CACHE_ALLOWED_STATUSES` = 200,301,302

### YAML

```yaml
server:
  port: "8081"
  ttl: 0
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

## TODO

  - Healthcheck Endpoint
  - Support Chunking
  - Serve STALE cache
  - LB Algorithms
  - Cache Circuit Breaker
  - Cache Backends: Redis, [BigCache](https://github.com/allegro/bigcache), [FreeCache](https://github.com/coocood/freecache)
  - Purge
  - Tags
  - HTTPS
  - HTTP/2
  - HTTP/3
  - WebSockets
  - Improve Logging
  - Define eviction: LRU, LFU, ...
  - Byte-Range Cache
  - Dashboard
  - CLI Monitor

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
