# Gcloud

## Cloud SQL (mysql)

connect to a DB using `gcloud`, opening firewall and starting `mysql` CLI:

```bash
gcloud sql connect <db_name> --user=<admin user> --project <gcloud project>
```

## Secrets

```bash
# list secrets
gcloud secrets list --project=<gcloud project>

# Backup secret
gcloud secrets versions access latest --secret=<secret name> --project=<gcloud project> --format='get(payload.data)' > /tmp/secret

# Copy backup into new secret
cat /tmp/secret | base64 -D | gcloud secrets versions add <secret name> --project=<gcloud project> --data-file=-

# view secret
gcloud secrets versions access latest --secret=<secret name> --project=<gcloud project>
```

## Workload Identity

- create a Pod with `gcloud` commands, and use the KSA that is supposed to have Workload Identity enabled

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: workload-identity-test
  namespace: NAMESPACE
spec:
  containers:
  - image: google/cloud-sdk:slim
    name: workload-identity-test
    command: ["sleep","infinity"]
  serviceAccountName: KSA_NAME
  nodeSelector:
    iam.gke.io/gke-metadata-server-enabled: "true"
```

- Exec into this pod and run a curl command to the metadata server

```bash
curl -H "Metadata-Flavor: Google" http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/email

curl  -H "Metadata-Flavor: Google" 'http://169.254.169.254/computeMetadata/v1/instance/service-accounts/<previous email>/email'
```

- Add more tooling

```bash
apt update
cd
wget https://dl.google.com/go/go1.24.1.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.24.1.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
go version
```
