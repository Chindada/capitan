#!/bin/bash
set -e

if [ -d certs ]; then
    echo "certs directory already exists, skip creating certs."
    exit 0
fi

mkdir -p certs

openssl req -nodes -new -x509 -keyout certs/ca.key -out certs/ca.crt -days 7300 -config scripts/cnf/ca.cnf
openssl req -sha256 -nodes -newkey rsa:2048 -keyout certs/key.pem -out certs/localhost.csr -config scripts/cnf/localhost.cnf
openssl x509 -req -days 7300 -in certs/localhost.csr -CA certs/ca.crt -CAkey certs/ca.key -CAcreateserial -out certs/cert.pem -extensions req_ext -extfile scripts/cnf/localhost.cnf

rm certs/ca.crt
rm certs/ca.srl
rm certs/ca.key
rm certs/localhost.csr

openssl dhparam -out certs/capitan.dhparam 2048
