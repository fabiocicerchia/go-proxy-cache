# Development

## Need tools

- Go v1.15
- make
- [wrk](https://github.com/wg/wrk)

## Monitor file descriptors

Launch wrk then:

```
lsof -p PID | wc -l
```
