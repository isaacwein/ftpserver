#!/bin/bash

# Generate a private key
openssl genrsa -out localhost.rsa.key 2048

# Generate a CSR using the configuration file
openssl req -new -key localhost.rsa.key -out localhost.rsa.csr -config localhost.rsa.cnf

# Generate a self-signed certificate using the CSR and private key
openssl x509 -req -in localhost.rsa.csr -signkey localhost.rsa.key -out localhost.rsa.crt

# Verify the certificate
openssl x509 -noout -text -in localhost.rsa.crt
