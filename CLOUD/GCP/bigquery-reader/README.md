# GCS Downloader

Download a file from a GCS bucket

Equivalent to:

```bash
curl -v  -X GET -H "Authorization: Bearer $(gcloud auth print-access-token)" -o ~/Downloads/GCS/db-data-volume.tar.gz "https://storage.googleapis.com/storage/v1/b/<BUCKET>/o/<file>?alt=media"
```