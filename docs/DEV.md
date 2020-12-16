# Development

## Need tools

- Go v1.15
- make
- [wrk](https://github.com/wg/wrk)

## Setup

```
echo "127.0.0.1 testing.local www.testing.local" >> /etc/hosts
cd examples
./gen-selfsigned-cert.sh*
docker-compose up
```

## Test

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

```
lsof -p PID | wc -l
```
