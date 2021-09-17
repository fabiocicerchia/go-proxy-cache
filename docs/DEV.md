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

Jaeger is available by using the `test/full-setup` demo and browsing to `http://127.0.0.1:16686/`.

Prometheus endpoint is available at `http://127.0.0.1:52021/metrics`. Its metrics are collected and available via Grafana
at `http://localhost:3001`.

There is a JSON export of the dashboard stored in `test/full-setup/grafana/gpc-dashboard.json`.

![GPC Grafana Dashboard](grafana.png)

Note: the Data Source must be configured in Grafana to point to `http://prometheus:9090`.

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
