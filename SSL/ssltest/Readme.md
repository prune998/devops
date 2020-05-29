# SSLTest

Project forked from https://github.com/ChristopherSchultz/ssltest

Use this app to test supported Ciphers and keystores

```bash
Usage: java -jar ssltest.jar [opts] host[:port]

-sslprotocol                 Sets the SSL/TLS protocol to be used (e.g. SSL, TLS, SSLv3, TLSv1.2, etc.)
-enabledprotocols protocols  Sets individual SSL/TLS ptotocols that should be enabled
-ciphers cipherspec          A comma-separated list of SSL/TLS ciphers
-keystore                    Sets the key store for connections (for TLS client certificates)
-keystoretype type           Sets the type for the key store
-keystorepassword pass       Sets the password for the key store
-keystoreprovider provider   Sets the crypto provider for the key store
-truststore                  Sets the trust store for connections
-truststoretype type         Sets the type for the trust store
-truststorepassword pass     Sets the password for the trust store
-truststorealgorithm alg     Sets the algorithm for the trust store
-truststoreprovider provider Sets the crypto provider for the trust store
-no-check-certificate        Ignores certificate errors
-no-verify-hostname          Ignores hostname mismatches
-showsslerrors               Show SSL/TLS error details
-showhandshakeerrors         Show SSL/TLS handshake error details
-showerrors                  Show all connection error details
-hiderejects                 Only show protocols/ciphers which were successful
```