#                                                                         __
# .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
# |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
# |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
# |_____|            |__|                   |_____|
#
# Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
# Repo: https://github.com/fabiocicerchia/go-proxy-cache

FROM golang:1.15.6-alpine3.12 AS builder

ARG BUILD_CMD=build

WORKDIR /go/src/github.com/fabiocicerchia/go-proxy-cache

ENV CGO_CFLAGS -march=native -O3

COPY . ./

RUN apk update \
    && apk add --no-cache \
      gcc \
      libc-dev \
      make \
    && make $BUILD_CMD

FROM alpine:3.12.3

WORKDIR /app

COPY --from=builder /go/src/github.com/fabiocicerchia/go-proxy-cache/go-proxy-cache /usr/local/bin/
COPY --from=builder /go/src/github.com/fabiocicerchia/go-proxy-cache/config.yml /app/

CMD ["go-proxy-cache"]
