# gen-cert.sh

This script create a self signed SSL Certificate (+ CA certificate) that is fully compatible with latest requirements (V3 SSL Cert). It can be used for a web server.
While you will still get a warning as it is a self-signed cert, it support alternate DNS names.

## Usage

`gen-cert.sh <DNS name> <destination folder> [<other DNS name>...]`

ex:

```bash
./gen-cert.sh my.app.com /tmp other.app.com
```
