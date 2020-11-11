FROM golang:1.15.4-alpine3.12 AS builder

WORKDIR /go/src/github.com/fabiocicerchia/go-proxy-cache

COPY . ./

RUN go build -o go-proxy-cache main.go

FROM alpine:3.12.1

COPY --from=builder /go/src/github.com/fabiocicerchia/go-proxy-cache/go-proxy-cache /usr/local/bin/

CMD ["go-proxy-cache"]