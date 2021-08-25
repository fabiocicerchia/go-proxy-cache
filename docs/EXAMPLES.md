# Examples

## CLI

```console
$ go-proxy-cache -h
Usage of go-proxy-cache:
  -config string
        config file (default "config.yml")
  -debug
        enable debug
  -log string
        log file (default stdout)
  -test
        test configuration
  -verbose
        enable verbose
  -version
        display version
[...]
```

## Docker

```console
$ docker run \
    -it --rm --name goproxycache \
    --env SERVER_HTTPS_PORT=443 \
    --env SERVER_HTTP_PORT=80 \
    --env DEFAULT_TTL=0 \
    --env FORWARD_HOST=www.google.com \
    --env FORWARD_SCHEME=https \
    --env LB_ENDPOINT_LIST=www.google.com \
    --env REDIS_DB=0 \
    --env REDIS_HOST=localhost \
    --env REDIS_PORT=6379 \
    --env REDIS_PASSWORD= \
    -p 8080:80 \
    -p 8443:443 \
    fabiocicerchia/go-proxy-cache
```

```console
curl -H"Host: www.google.com" -v http://127.0.0.1:8080/
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to 127.0.0.1 (127.0.0.1) port 8080 (#0)
> GET / HTTP/1.1
> Host: www.google.com
> User-Agent: curl/7.64.1
> Accept: */*
>
< HTTP/1.1 301 Moved Permanently
< Content-Type: text/html; charset=utf-8
< Location: https://www.google.com/
< Date: Wed, 25 Aug 2021 13:30:19 GMT
< Content-Length: 58
<
<a href="https://www.google.com/">Moved Permanently</a>.

* Connection #0 to host 127.0.0.1 left intact
* Closing connection 0
```

## PURGE

```concole
$ curl -vX PURGE http://localhost/cached/page
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to localhost (127.0.0.1) port 80 (#0)
> PURGE / HTTP/1.1
> Host: localhost
> User-Agent: curl/7.64.1
> Accept: */*
>
< HTTP/1.1 200 OK
< Date: Thu, 19 Nov 2020 11:21:45 GMT
< Content-Length: 2
< Content-Type: text/plain; charset=utf-8
<
* Connection #0 to host localhost left intact
OK* Closing connection 0
```

```concole
$ curl -vX PURGE http://localhost/page/not/cached
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to localhost (127.0.0.1) port 80 (#0)
> PURGE / HTTP/1.1
> Host: localhost
> User-Agent: curl/7.64.1
> Accept: */*
>
< HTTP/1.1 404 Not Found
< Date: Thu, 19 Nov 2020 11:23:36 GMT
< Content-Length: 2
< Content-Type: text/plain; charset=utf-8
<
* Connection #0 to host localhost left intact
KO* Closing connection 0
```

## HTTP/2

```console
$ curl -4 -s -I -w '%{http_version}\n' -o /dev/null http://localhost
1.1
$ curl -4 -k -s -I -w '%{http_version}\n' -o /dev/null https://localhost
2
```

## HealthCheck

```console
$ curl -v http://localhost/healthcheck
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to localhost (127.0.0.1) port 80 (#0)
> GET /healthcheck HTTP/1.1
> Host: localhost
> User-Agent: curl/7.64.1
> Accept: */*
>
< HTTP/1.1 200 OK
< Date: Thu, 19 Nov 2020 11:26:37 GMT
< Content-Length: 17
< Content-Type: text/plain; charset=utf-8
<
HTTP OK
REDIS OK
* Connection #0 to host localhost left intact
* Closing connection 0
```
