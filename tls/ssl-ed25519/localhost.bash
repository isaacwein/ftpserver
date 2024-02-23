#!/bin/bash

set -x

openssl genpkey -algorithm ED25519 -out localhost.key


openssl req -new -key localhost.key -out localhost.csr -config localhost.cnf

openssl req -in localhost.csr -text -noout > localhost.csr.txt


openssl x509 -req -days 365 -in localhost.csr -signkey localhost.key -out localhost.crt



