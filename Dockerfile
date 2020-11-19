FROM golang:1.15.5-alpine3.12 AS builder

WORKDIR /go/src/github.com/fabiocicerchia/go-proxy-cache

COPY . ./

RUN apk update \
    && apk add --no-cache \
      gcc \
      libc-dev \
      make \
      redis \
    && make build \
    && redis-server --daemonize yes \
    && make test

FROM alpine:3.12.1

WORKDIR /app

COPY --from=builder /go/src/github.com/fabiocicerchia/go-proxy-cache/go-proxy-cache /usr/local/bin/
COPY --from=builder /go/src/github.com/fabiocicerchia/go-proxy-cache/config.yml /app/

CMD ["go-proxy-cache"]
