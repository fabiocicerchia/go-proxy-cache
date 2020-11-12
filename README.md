# Go Proxy Cache

[![MIT License](https://img.shields.io/badge/License-MIT-lightgrey.svg?longCache=true)](LICENSE)
[![Pull Requests](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?longCache=true)](https://github.com/fabiocicerchia/go-proxy-cache/pulls)
![Last Commit](https://img.shields.io/github/last-commit/fabiocicerchia/go-proxy-cache)
![Release Date](https://img.shields.io/github/release-date/fabiocicerchia/go-proxy-cache)

![Docker pulls](https://img.shields.io/docker/pulls/fabiocicerchia/go-proxy-cache.svg "Docker pulls")
![Docker stars](https://img.shields.io/docker/stars/fabiocicerchia/go-proxy-cache.svg "Docker stars")
![Known Vulnerabilities](https://img.shields.io/badge/vulnerabilities-snyk-4b45a9)

![Docker](https://github.com/fabiocicerchia/go-proxy-cache/workflows/Docker/badge.svg)
![Docker Builds](https://github.com/fabiocicerchia/go-proxy-cache/workflows/Docker%20Builds/badge.svg)

Simple caching proxy written in golang backed by redis.

## Features

 - HTTP Forward Traffic
 - Full Page Caching (via Redis)
 - Small, pragmatic and easy to use
 - Self-contained, does not require Go, Git or any other software to be installed. Just run the binary.

## Docker

```console
$ docker run \
    -it --rm -n goproxycache \
    --env SERVER_PORT=8080 \
    --env FORWARD_TO=https://www.google.com \
    --env DEFAULT_TTL=0 \
    --env REDIS_HOST=localhost \
    --env REDIS_PORT=6379 \
    --env REDIS_PASSWORD= \
    --env REDIS_DB=0 \
    fabiocicerchia/go-proxy-cache
```

## TODO

 - Load Balancing
 - SSL Termination
 - Improve Logging
 - Configuration File
 - Testing

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
