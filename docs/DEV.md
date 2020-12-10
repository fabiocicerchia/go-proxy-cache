# Development

## Need tools

- Go v1.15
- make
- [wrk](https://github.com/wg/wrk)

## Setup

```
echo "127.0.0.1 testing.local www.testing.local" >> /etc/hosts
cd examples
docker-compose up
```

## Monitor file descriptors

Launch wrk then:

```
lsof -p PID | wc -l
```
