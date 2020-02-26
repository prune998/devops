#!/bin/bash

set -ex

CN=${1:?primary DNS name is required}
OUT=${2:?certificate output directory is required}
ALTNAMES=${3}

mkdir -p ${OUT}

# Generate the config for OpenSSL. This ensure a working cert on any OS
# we do this mainly because of some bugs in openssl : https://security.stackexchange.com/questions/150078/missing-x509-extensions-with-an-openssl-generated-certificate
cat > ${OUT}/openssl.conf <<EOF
[ req ]
req_extensions = v3_req
distinguished_name = req_distinguished_name
x509_extensions = usr_cert
x509_extensions = v3_ca
copy_extensions = copyall
prompt = no

[req_distinguished_name]
countryName            = US
stateOrProvinceName    = CA
localityName           = SF
organizationName       = NONE
organizationalUnitName = NONE.IO
commonName             = ${CN}

[usr_cert]
basicConstraints = CA:FALSE
nsCertType = client, server, email
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth, clientAuth, emailProtection
nsComment=KeyTalk Client Cert
subjectKeyIdentifier=hash
authorityKeyIdentifier=keyid,issuer

[v3_req]
basicConstraints = CA:FALSE
subjectAltName = @alt_names
extendedKeyUsage = serverAuth, clientAuth, emailProtection
keyUsage=nonRepudiation, digitalSignature, keyEncipherment

[v3_ca]
basicConstraints = CA:FALSE
extendedKeyUsage = serverAuth, clientAuth, emailProtection
keyUsage=nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = ${CN}
EOF

cat > ${OUT}/ssl-extensions-x509.cnf <<EOF
[v3_ca]
basicConstraints = CA:FALSE
subjectAltName = @alt_names
extendedKeyUsage = serverAuth, clientAuth, emailProtection
keyUsage=nonRepudiation, digitalSignature, keyEncipherment

[alt_names]
DNS.1 = ${CN}
EOF

# add Altername Name to the setup file
if [ "$#" -gt 2 ]; then
	echo "adding other arguments as Alternate Names"
  shift 2
  CUR=2
  while (( "$#" )); do
    echo "DNS.${CUR} = ${1}" >> ${OUT}/openssl.conf
    echo "DNS.${CUR} = ${1}" >> ${OUT}/ssl-extensions-x509.cnf
    ((++CUR))
    shift
  done
fi

# Setup the CA
openssl genrsa -out ${OUT}/${CN}-ca.key 4096
openssl req -x509 -new -nodes -key ${OUT}/${CN}-ca.key -config ${OUT}/openssl.conf  -sha256 -days 1024 -out ${OUT}/${CN}-ca.crt -verbose

# Create the app certificate
openssl genrsa -out ${OUT}/${CN}.key 2048
openssl req -config ${OUT}/openssl.conf -new -key ${OUT}/${CN}.key -out ${OUT}/${CN}.csr -verbose
openssl x509 -req -extensions v3_ca  -extfile ${OUT}/ssl-extensions-x509.cnf -in ${OUT}/${CN}.csr -CA ${OUT}/${CN}-ca.crt -CAkey ${OUT}/${CN}-ca.key -CAcreateserial -out ${OUT}/${CN}.crt -days 500 -sha256

# display resulting cert
openssl req -text -noout -verify -in ${OUT}/${CN}.csr
openssl x509 -in ${OUT}/${CN}.crt -text

# remove build files
rm -f ${OUT}/openssl.conf ${OUT}/ssl-extensions-x509.cnf