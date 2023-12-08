# Docker

## Official Image

Available on Docker Hub: [https://hub.docker.com/r/fabiocicerchia/go-proxy-cache](https://hub.docker.com/r/fabiocicerchia/go-proxy-cache)

![Docker pulls](https://img.shields.io/docker/pulls/fabiocicerchia/go-proxy-cache.svg "Docker pulls")
![Docker stars](https://img.shields.io/docker/stars/fabiocicerchia/go-proxy-cache.svg "Docker stars")

## CLI

Example #1:

```console
$ docker run \
    -it --rm -n goproxycache \
    --env SERVER_HTTPS_PORT=443 \
    --env SERVER_HTTP_PORT=80 \
    --env DEFAULT_TTL=0 \
    --env FORWARD_HOST=www.google.com \
    --env FORWARD_SCHEME=https \
    --env LB_ENDPOINT_LIST=www.google.com \
    --env REDIS_DB=0 \
    --env REDIS_HOSTS=localhost:6379 \
    --env REDIS_PASSWORD= \
    -p 80:80
    -p 443:443
    fabiocicerchia/go-proxy-cache
```

Example #2:

```console
$ docker run \
    -it --rm -n goproxycache \
    -v $PWD/config.yml:/app/config.yml
    -p 80:80
    -p 443:443
    fabiocicerchia/go-proxy-cache
```

## Docker Compose

```yaml
version: '3.7'

services:
  goproxycache:
    image: fabiocicerchia/go-proxy-cache:latest
    restart: always
    network_mode: host
    volumes:
      - ./config.yml:/app/config.yml

  redis:
    image: redis:alpine
    restart: always
    ports:
      - "6379:6379"

  [...]
```
