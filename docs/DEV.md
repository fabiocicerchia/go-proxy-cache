# Development

## Need tools

- Go v1.15
- make
- [wrk](https://github.com/wg/wrk)

## Setup

```console
$ docker build -t fabiocicerchia/go-proxy-cache-test:latest -f examples/Dockerfile examples
$ echo "127.0.0.1 testing.local www.testing.local" >> /etc/hosts
$ cd examples
$ ./gen-selfsigned-cert.sh
$ docker-compose up
```

## Test

**NOTE:** If you use docker please use `config.yml` otherwise `config.no-docker.yml`. The port will be different from host and container, this will address the issue.

```console
$ make test
[...]
$ cd examples && node ws_client.js
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
