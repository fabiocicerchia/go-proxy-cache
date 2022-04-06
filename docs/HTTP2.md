# HTTP/2

## Push

The functionality is deprecated since not really supported in the browsers.

More details:
 - [Current implementation](https://www.w3.org/TR/preload/)
 - [What it was supposed to be like](https://medium.com/@mena.meseha/http-2-server-push-tutorial-d8714154ef9a)
 - HTTP/2 Push is dead:
   - [Chrome to remove HTTP/2 Push](https://www.ctrl.blog/entry/http2-push-chromium-deprecation.html)
   - [Google Developers intent to remove HTTP/2 Push](https://community.cloudflare.com/t/google-developers-intent-to-remove-http-2-push/261338)
   - [Intent to Remove: HTTP/2 and gQUIC server push](https://groups.google.com/a/chromium.org/g/blink-dev/c/K3rYLvmQUBY/m/vOWBKZGoAQAJ)
   - [HTTP/2 Push is dead](https://evertpot.com/http-2-push-is-dead/)
 - Alternative: [103 Early Hints](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/103)

### Upstream

```nginx
# ...

server {
    # ...

    location = /push {
        add_header Content-Type text/plain;
        add_header Cache-Control "public, max-age=86400";
        add_header Link "</etag>; rel=preload";
        default_type text/plain;
        return 200 "push";
    }

    # ...
}

# ...
```

### Test

You can use the `nghttp` cli tool ([nghttp2.org](https://nghttp2.org/)) project to verify whether the server push is working.

```console
nghttp -ans https://testing.local:50443/push
[WARNING] Certificate verification failed: unable to verify the first certificate
***** Statistics *****

Request timing:
responseEnd: the  time  when  last  byte of  response  was  received relative to connectEnd
requestStart: the time  just before  first byte  of request  was sent relative  to connectEnd.   If  '*' is  shown, this  was pushed by server.
process: responseEnd - requestStart
code: HTTP status code
size: number  of  bytes  received as  response  body  without inflation.
URI: request URI

see http://www.w3.org/TR/resource-timing/#processing-model

sorted by 'complete'

id  responseEnd requestStart  process code size request path
  13    +41.68ms       +134us  41.54ms  200    4 /push
  2     +86.95ms *   +37.31ms  49.64ms  200    4 /etag
```

In the output, the asterisk (`*`) marks resources that were pushed by the server.
