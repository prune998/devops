# HTTPSClient

Both apps connect to an httpS server to validate java can negociate protocol and ciphers

## JKS Store
If you have a PEM encoded certificate (CA and Intermediate), create a JKS store:

```bash
for file in ca.crt intermediate.crt; do keytool -import -noprompt -keystore trust.jks -file $file -storepass mypass -alias service-$file; done
keytool -list -keystore trust.jks -storepass mypass
```

## HTTPSClient1
```bash
javac HTTPSClient.java
java HTTPSClient
```

## HTTPSClient2

Edit the `URL` value from `HTTPSClient2.java`. It must be like `https://user:pass@my_url:port`

```bash
javac HTTPSClient2.java
java HTTPSClient2
```