# Benchmark

## Configuration

Check config files `docker-compose.yml` and `config.benchmark.yml` in [`benchmark`](benchmark) folder.

## Proxied vs Direct

direct:

```console
$ wrk -t12 -c1000 -d30s -H "Host: www.w3.org" http://127.0.0.1:81
Running 30s test @ http://127.0.0.1:81
  12 threads and 1000 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    44.89ms   34.09ms 476.69ms   94.08%
    Req/Sec   622.00    411.92     1.87k    61.58%
  161618 requests in 30.09s, 131.00MB read
  Socket errors: connect 755, read 147, write 0, timeout 0
Requests/sec:   5371.19
Transfer/sec:      4.35MB
```

proxied:

```console
$ wrk -t12 -c1000 -d30s -H "Host: www.w3.org" http://127.0.0.1:80
Running 30s test @ http://127.0.0.1:80
  12 threads and 1000 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     1.59s   316.07ms   2.00s    74.44%
    Req/Sec    22.14     20.74   168.00     79.36%
  4064 requests in 30.10s, 3.32MB read
  Socket errors: connect 755, read 116, write 0, timeout 1040
Requests/sec:    135.01
Transfer/sec:    112.99KB
```

---

## Non valid domain

Non valid domain for 30s:

```console
$ ./go-proxy-cache -config config.sample.yml &
$ wrk -t12 -c1000 -d30s http://127.0.0.1:8080
Running 30s test @ http://127.0.0.1:8080
  12 threads and 1000 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency   349.95ms  592.11ms   1.80s    84.21%
    Req/Sec    26.43     27.14   222.00     85.22%
  4083 requests in 30.10s, 350.88KB read
  Socket errors: connect 0, read 4637, write 0, timeout 4064
  Non-2xx or 3xx responses: 4083
Requests/sec:    135.63
Transfer/sec:     11.66KB
```

Non valid domain for 120s:

```console
$ ./go-proxy-cache -config config.sample.yml &
$ wrk -t12 -c1000 -d120s http://127.0.0.1:8080
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency   625.42ms  140.88ms   2.00s    90.13%
    Req/Sec   119.35    124.17   626.00     81.62%
  113185 requests in 2.00m, 10.02MB read
  Socket errors: connect 0, read 4343, write 0, timeout 10779
  Non-2xx or 3xx responses: 113185
Requests/sec:    942.41
Transfer/sec:     85.41KB
```

## Valid domain without redis

Valid domain without redis for 30s:

```console
$ ./go-proxy-cache -config config.sample.yml &
$ wrk -t12 -c1000 -d30s -H"Host: fabiocicerchia.it" http://127.0.0.1:8080
Running 30s test @ http://127.0.0.1:8080
  12 threads and 1000 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency   430.99ms  365.78ms   2.00s    89.07%
    Req/Sec   124.77    137.97   626.00     80.84%
  35961 requests in 30.10s, 7.89MB read
  Socket errors: connect 0, read 3823, write 0, timeout 5792
Requests/sec:   1194.86
Transfer/sec:    268.38KB
```

Valid domain without redis for 120s:

```console
$ ./go-proxy-cache -config config.sample.yml &
$ wrk -t12 -c1000 -d120s -H"Host: fabiocicerchia.it" http://127.0.0.1:8080
Running 2m test @ http://127.0.0.1:8080
  12 threads and 1000 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency   354.84ms  185.14ms   2.00s    95.76%
    Req/Sec    94.40    126.95   696.00     83.61%
  108464 requests in 2.00m, 23.79MB read
  Socket errors: connect 0, read 3813, write 0, timeout 24928
Requests/sec:    903.27
Transfer/sec:    202.88KB
```

## Valid domain uncacheable with redis

Valid domain uncacheable with redis for 30s:

```console
$ ./go-proxy-cache -config config.sample.yml &
$ wrk -t12 -c1000 -d30s -H"Host: fabiocicerchia.it" http://127.0.0.1:8080
Running 30s test @ http://127.0.0.1:8080
  12 threads and 1000 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency   681.59ms  586.78ms   2.00s    80.54%
    Req/Sec    72.20     96.33   686.00     88.91%
  20461 requests in 30.10s, 4.49MB read
  Socket errors: connect 0, read 3926, write 0, timeout 7064
Requests/sec:    679.73
Transfer/sec:    152.67KB
```

## Valid domain with redis

Valid domain with redis for 120s:

```console
$ ./go-proxy-cache -config config.sample.yml &
$ wrk -t12 -c1000 -d120s -H"Host: fabiocicerchia.it" http://127.0.0.1:8080
Running 2m test @ http://127.0.0.1:8080
  12 threads and 1000 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency   471.05ms  365.04ms   2.00s    91.64%
    Req/Sec   100.72    110.76   575.00     81.43%
  121761 requests in 2.00m, 26.71MB read
  Socket errors: connect 0, read 3791, write 0, timeout 23590
Requests/sec:   1013.96
Transfer/sec:    227.75KB
```

---

Valid domain with redis for 120s:

```console
$ ./go-proxy-cache -config config.sample.yml &
$ wrk -t12 -c1000 -d120s -H"Host: www.w3.org" https://127.0.0.1:8443
Running 2m test @ https://127.0.0.1:8443
  12 threads and 1000 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     0.00us    0.00us   0.00us     nan%
    Req/Sec     0.00      0.00     0.00       nan%
  0 requests in 2.00m, 0.00B read
  Socket errors: connect 239606, read 0, write 0, timeout 0
Requests/sec:      0.00
Transfer/sec:       0.00B
```

## Valid domain cacheable cacheable with redis

Valid domain cacheable cacheable with redis for 30s:

```console
$ echo "127.0.0.1 www.w3.org" > /etc/hosts
$ ./go-proxy-cache -config config.sample.yml &
$ wrk -t12 -c1000 -d30s https://www.w3.org:8443
Running 30s test @ https://www.w3.org:8443
  12 threads and 1000 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     0.00us    0.00us   0.00us     nan%
    Req/Sec    41.54     27.03   151.00     66.16%
  4980 requests in 30.10s, 700.31KB read
  Socket errors: connect 4323, read 23, write 0, timeout 4980
  Non-2xx or 3xx responses: 4980
Requests/sec:    165.47
Transfer/sec:     23.27KB
```

Valid domain uncacheable cacheable with redis for 30s:

```console
$ ./go-proxy-cache -config config.sample.yml &
$ wrk -t12 -c1000 -d30s -s benchmark/script.lua http://127.0.0.1:8080 
Running 30s test @ http://127.0.0.1:8080
  12 threads and 1000 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     1.81s     0.00us   1.81s   100.00%
    Req/Sec    50.85     48.16   287.00     68.71%
  5262 requests in 30.10s, 8.97MB read
  Socket errors: connect 0, read 5683, write 40, timeout 5261
  Non-2xx or 3xx responses: 4964
Requests/sec:    174.80
Transfer/sec:    304.97KB
```

Valid domain cacheable with redis for 30s:

```console
$ ./go-proxy-cache -config config.sample.yml &
$ wrk -t12 -c1000 -d30s https://www.w3.org:8443/standards/
Running 30s test @ https://www.w3.org:8443/standards/
  12 threads and 1000 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     1.83s    76.26ms   1.90s    75.00%
    Req/Sec    36.21     33.41   198.00     65.11%
  2000 requests in 30.09s, 17.18MB read
  Socket errors: connect 903, read 0, write 0, timeout 1996
  Non-2xx or 3xx responses: 996
Requests/sec:     66.47
Transfer/sec:    584.56KB
```


