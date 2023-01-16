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