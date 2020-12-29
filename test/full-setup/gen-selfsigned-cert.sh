#!/usr/bin/env bash

# CA
openssl req \
  -new \
  -x509 \
  -days 9999 \
  -config certs/ca.cnf \
  -keyout certs/ca-key.pem \
  -out certs/ca-crt.pem

openssl genrsa -out certs/server.key 4096

# CSR
openssl req \
  -new \
  -config certs/server.cnf \
  -key certs/server.key \
  -out certs/server.csr

# CERT
openssl x509 \
  -req \
  -extfile certs/server.cnf \
  -days 3650 \
  -passin "pass:password" \
  -in certs/server.csr \
  -CA certs/ca-crt.pem \
  -CAkey certs/ca-key.pem \
  -CAcreateserial \
  -out certs/server.pem
