#!/usr/bin/env bash

DOMAIN=$1
if [ "$DOMAIN" == "" ]; then
  DOMAIN="default"
fi

cd certs/$DOMAIN

# CA
openssl req \
  -new \
  -x509 \
  -days 9999 \
  -config ../ca.cnf \
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
