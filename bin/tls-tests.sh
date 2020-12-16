#!/bin/sh

# Ref: https://www.yeahhub.com/testing-methods-https-openssl-curl-nmap/

# OPENSSL
openssl s_client -connect localhost:443

# OPTIONS
curl --ipv4 --include --request OPTIONS http://localhost/
curl --ipv4 --insecure --include --request OPTIONS https://localhost/

# NMAP
nmap -script http-methods -p80 -script-args http-methods.url-path='/' localhost
nmap -script http-methods -p443 -script-args http-methods.url-path='/' localhost

# HTTP PROTOCOL VERSION
curl --ipv4 --insecure --silent --head --write-out '%{http_version}\n' --output /dev/null http://localhost
curl --ipv4 --insecure --silent --head --write-out '%{http_version}\n' --output /dev/null https://localhost

# CURL VARIABLES
curl --ipv4 --include --verbose --write-out @curl_vars.txt --http1.1 http://localhost/ 2>&1
curl --ipv4 --insecure --include --verbose --write-out @curl_vars.txt --http1.1 https://localhost/ 2>&1
curl --ipv4 --insecure --include --verbose --write-out @curl_vars.txt --http2 https://localhost/ 2>&1

# PROXY TESTS
curl --ipv4 --insecure --verbose --proxy http://localhost:80 http://testing.local
curl --ipv4 --insecure --proxy-insecure --verbose --proxy https://localhost:443 https://testing.local

# WEBSOCKET
curl --include \
     --no-buffer \
     --header "Connection: Upgrade" \
     --header "Upgrade: websocket" \
     --header "Host: example.com:80" \
     --header "Origin: http://testing.local:80" \
     --header "Sec-WebSocket-Key: SGVsbG8sIHdvcmxkIQ==" \
     --header "Sec-WebSocket-Version: 13" \
     http://testing.local:80/

# WEBSOCKET SECURE
curl --include \
     --no-buffer \
     --header "Connection: Upgrade" \
     --header "Upgrade: websocket" \
     --header "Host: example.com:80" \
     --header "Origin: https://testing.local:443" \
     --header "Sec-WebSocket-Key: SGVsbG8sIHdvcmxkIQ==" \
     --header "Sec-WebSocket-Version: 13" \
     https://testing.local:443/
