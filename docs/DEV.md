# Development

## Need tools

- Go v1.15
- make
- [wrk](https://github.com/wg/wrk)

## Setup

```console
$ docker build -t fabiocicerchia/go-proxy-cache-test:nginx -f test/full-setup/Dockerfile.nginx test/full-setup
$ docker build -t fabiocicerchia/go-proxy-cache-test:node -f test/full-setup/Dockerfile.node test/full-setup
$ echo "127.0.0.1 testing.local www.testing.local" >> /etc/hosts
$ cd test/full-setup
$ ./gen-selfsigned-cert.sh
$ ./gen-selfsigned-cert.sh www.w3.org
$ docker-compose up
```

## Tracing

Jaeger is available by using the `test/full-setup` demo and browsing to `http://localhost:16686/`.

## Test

**NOTE:** In order to have a fully working environment you need to put in the host file `127.0.0.1 nginx`.

```console
$ make test
[...]
$ cd test/full-setup && node ws_client.js
launched plain
launched secure
Sending plain message
Server received from client: {}
Sending secure message
Server received from client: {}
^C
```

## Monitor file descriptors

Launch wrk then:

```console
$ lsof -p PID | wc -l
```
