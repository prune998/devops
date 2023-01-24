# GSM Editor

Edit one value in a YAML Google Secret


Before use, configure `gcloud` to allow your default login for `go` apps:

```bash
gcloud auth application-default login
```

Equivalent to:

```bash
google secrets <get whole secret> > file
sed file
google secrets <upload new version> file 
```

In fact, you could do all that with `yq`... 

The secret needs to be YAML encoded.
Ex secret:

```yaml
thanos:
  objstore:
    config:
      service_account: |-
        {
          "type": "service_account",
          "project_id": "my-project",
          "private_key_id": "123456789",
          "client_email": "thanos-bucket-user@my-project.iam.gserviceaccount.com",
          "client_id": "123456789",
          "auth_uri": "https://accounts.google.com/o/oauth2/auth",
          "token_uri": "https://oauth2.googleapis.com/token",
          "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
          "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/thanos-bucket-user%40my-project.iam.gserviceaccount.com"
        }
```

you can replace the `service_account` part with:

```bash
./gsm-editor \
  -projectName=my-project \
  -secretKey=/thanos/objstore/config/service_account \
  -secretName=my-secret \
  -logLevel=debug \
  -secretValue='"new value"'
```

If you use multiline, remember to indent as you want it for the final result, not for the commandeline:

```bash
./gsm-editor \
  -projectName=my-project \
  -secretKey=/thanos/objstore/config/service_account \
  -secretName=my-secret \
  -logLevel=debug \
  -secretValue='{
  "type": "service_account",
  "project_id": "my-project",
  "private_key_id": "123456789",
  "client_email": "thanos-bucket-user@my-project.iam.gserviceaccount.com",
  "client_id": "123456789",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/thanos-bucket-user%40my-project.iam.gserviceaccount.com"
}'


Type `Y` to do the change or `-quiet` for automatic update. 