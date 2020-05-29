# CFSSL

Inspired by https://medium.com/@rob.blackbourn/how-to-use-cfssl-to-create-self-signed-certificates-d55f76ba5781

## Install

Requires `Go`

```bash
go get -u github.com/cloudflare/cfssl/cmd/cfssl
go get -u github.com/cloudflare/cfssl/cmd/cfssljson
```

## Usage

Generate a `CA`

```bash
cfssl gencert -initca ca.json | cfssljson -bare ca
```

Generate an `Intermediate CA`

```bash
cfssl gencert -initca intermediate-ca.json | cfssljson -bare intermediate_ca
cfssl sign -ca ca.pem -ca-key ca-key.pem -config cfssl.json -profile intermediate_ca intermediate_ca.csr | cfssljson -bare intermediate_ca
```

Generate a `Certificate`

```bash
cfssl gencert -ca intermediate_ca.pem -ca-key intermediate_ca-key.pem -config cfssl.json -profile=peer   certificate.json | cfssljson -bare certificate-peer
cfssl gencert -ca intermediate_ca.pem -ca-key intermediate_ca-key.pem -config cfssl.json -profile=server certificate.json | cfssljson -bare certificate-server
cfssl gencert -ca intermediate_ca.pem -ca-key intermediate_ca-key.pem -config cfssl.json -profile=client certificate.json | cfssljson -bare certificate-client
```

Generate the `CA bundle`:

```bash
cat ca.pem intermediate_ca.pem > bundle-ca.pem
```