# Google Cloud Function Test

This is a sample Cloud Function

Example taken from:

- [github repo](https://github.com/GoogleCloudPlatform/golang-samples/blob/HEAD/functions/helloworld/hello_http.go)
- [google doc](https://cloud.google.com/functions/docs/create-deploy-http-go)

The fonctions was extended to generate a metric and call a Prometheus Push Gateway to push the metric to.

## Usage

Deploy the function with:

```bash
gcloud functions deploy HelloHTTP --runtime go120 --trigger-http --allow-unauthenticated --vpc-connector vpc-access-connector
```

Check the function is OK:

```bash
gcloud functions describe HelloHTTP
```

Then curl on the URL at `httpsTrigger.url` like `curl -kvs https://us-central1-<project>.cloudfunctions.net/HelloHTTP`.

You can pass a value using `curl -X POST  -kvs https://us-central1-<project>.cloudfunctions.net/HelloHTTP -H "Content-Type:application/json"  -d '{"name":"NAME"}'`

Check logs using `gcloud functions logs read HelloHTTP`