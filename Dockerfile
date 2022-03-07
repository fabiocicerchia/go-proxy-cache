#                                                                         __
# .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
# |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
# |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
# |_____|            |__|                   |_____|
#
# Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
# Repo: https://github.com/fabiocicerchia/go-proxy-cache

FROM golang:1.17.8-alpine3.14 AS builder

ARG BUILD_CMD=build

WORKDIR /go/src/github.com/fabiocicerchia/go-proxy-cache

ENV CGO_CFLAGS -march=native -O3

RUN apk update \
    && apk add --no-cache \
      gcc \
      libc-dev \
      make

COPY . ./

RUN make $BUILD_CMD

FROM alpine:3.14.2

WORKDIR /app

COPY --from=builder /go/src/github.com/fabiocicerchia/go-proxy-cache/go-proxy-cache /usr/local/bin/
COPY --from=builder /go/src/github.com/fabiocicerchia/go-proxy-cache/config.yml.dist /app/config.yml

CMD ["go-proxy-cache"]
