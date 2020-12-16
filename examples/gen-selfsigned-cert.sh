#!/usr/bin/env bash

case $(uname -s) in
Linux*) sslConfig=/etc/ssl/openssl.cnf ;;
Darwin*) sslConfig=/System/Library/OpenSSL/openssl.cnf ;;
esac

openssl req \
  -newkey rsa:2048 \
  -x509 \
  -nodes \
  -keyout server.key \
  -new \
  -out server.pem \
  -subj /CN=testing.local \
  -reqexts SAN \
  -extensions SAN \
  -config <(cat $sslConfig \
    <(printf '[SAN]\nsubjectAltName=DNS:www.testing.local')) \
  -sha256 \
  -days 3650
