#!/usr/bin/env bash

# case $(uname -s) in
# Linux*) sslConfig=/etc/ssl/openssl.cnf ;;
# Darwin*) sslConfig=/System/Library/OpenSSL/openssl.cnf ;;
# esac

# openssl req \
#   -newkey rsa:2048 \
#   -x509 \
#   -nodes \
#   -keyout server.key \
#   -new \
#   -out server.pem \
#   -subj /CN=testing.local \
#   -reqexts SAN \
#   -extensions SAN \
#   -config <(cat $sslConfig \
#     <(printf '[SAN]\nsubjectAltName=DNS:www.testing.local')) \
#   -sha256 \
#   -days 3650

# CA
openssl req \
  -new \
  -x509 \
  -days 9999 \
  -config ca.cnf \
  -keyout ca-key.pem \
  -out ca-crt.pem

openssl genrsa -out server.key 4096

# CSR
openssl req \
  -new \
  -config server.cnf \
  -key server.key \
  -out server.csr

# CERT
openssl x509 \
  -req \
  -extfile server.cnf \
  -days 3650 \
  -passin "pass:password" \
  -in server.csr \
  -CA ca-crt.pem \
  -CAkey ca-key.pem \
  -CAcreateserial \
  -out server.pem
