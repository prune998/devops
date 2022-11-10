# FAQ for Kubernetes

# Search things

## Find resources with Finalizers

It may happen that a resource stay pending on `Terminating` state. It usually means the resource is waiting for a `Finalizer` to complete. If the Finalizer is an Operator that you already removed, it will never complete (and remove) the Finilazer.
List resources with Finalizers in a namespace:

```bash
# example for istio-system namespace
kubectl api-resources --verbs=list --namespaced -o name   | xargs -n 1 kubectl get --show-kind --ignore-not-found --no-headers -o name -n istio-system

NAME                                                   AGE
istiooperator.install.istio.io/istiod   132m


kubectl -n istio-system edit istiooperator istiod
# Then delete the finalizer from the resource yaml and save
```

## List pods per UID

This is useful to find a pod when you know the UID as found in the filesystem of the node:

```bash
k --context gke_bx-production-ops_us-east4_prod-ops-cluster get pods -n ops -o custom-columns=PodName:.metadata.name,PodUID:.metadata.uid
```

# Istio

## Limit config sent to a Gateway

set `PILOT_FILTER_GATEWAY_CLUSTER_CONFIG` to `true`

## Start a debug pod with Istio injected

```bash
kubectl run -it --rm --labels="sidecar.istio.io/inject=true" --image=debian:latest
apt update && apt install -y dnsutils

dig +noall +answer xxx
...
```

# Custom Metrics

Custom metrics are generated when you install a metrics adapter. See [this blog post](https://medium.com/uptime-99/kubernetes-hpa-autoscaling-with-custom-and-external-metrics-da7f41ff7846).

## Check available APIs

```bash
k api-resources  -o wide | grep metric

NAME                                    SHORTNAMES                                                                         APIVERSION                                           NAMESPACED   KIND                                   VERBS
prometheus-query                                                                                                           external.metrics.k8s.io/v1beta1                      true         ExternalMetricValueList                [get]
logginglogmetrics                       gcplogginglogmetric,gcplogginglogmetrics                                           logging.cnrm.cloud.google.com/v1beta1                true         LoggingLogMetric                       [delete deletecollection get list patch create update watch]
nodes                                                                                                                      metrics.k8s.io/v1beta1                               false        NodeMetrics                            [get list]
pods                                                                                                                       metrics.k8s.io/v1beta1                               true         PodMetrics                             [get list]
monitoringmetricdescriptors             gcpmonitoringmetricdescriptor,gcpmonitoringmetricdescriptors                       monitoring.cnrm.cloud.google.com/v1beta1             true         MonitoringMetricDescriptor             [delete deletecollection get list patch create update watch]
```

## check which Service answer an API

```bash
 k get APIService | grep metric

NAME                                   SERVICE                      AVAILABLE   AGE
v1beta1.custom.metrics.k8s.io          ops/kube-metrics-adapter     True        638d
v1beta1.external.metrics.k8s.io        ops/kube-metrics-adapter     True        638d
v1beta1.metrics.k8s.io                 kube-system/metrics-server   True        3y278d
```

## list Nodes metrics

```bash
kubectl get --raw "/apis/metrics.k8s.io/v1beta1/nodes"|jq '.'

{
  "kind": "NodeMetricsList",
  "apiVersion": "metrics.k8s.io/v1beta1",
  "metadata": {},
  "items": [
    {
      "metadata": {
        "name": "gke-qa-us-central--auto-reply-pool-ad94ddd8-f132",
        "creationTimestamp": "2022-10-04T13:35:39Z",
        "labels": {
          "beta.kubernetes.io/arch": "amd64",
          "beta.kubernetes.io/instance-type": "e2-standard-2",
          "beta.kubernetes.io/os": "linux",
          "cloud.google.com/gke-boot-disk": "pd-ssd",
          "cloud.google.com/gke-container-runtime": "containerd",
          "cloud.google.com/gke-cpu-scaling-level": "2",
          "cloud.google.com/gke-max-pods-per-node": "110",
          "cloud.google.com/gke-netd-ready": "true",
          "cloud.google.com/gke-nodepool": "auto-reply-pool",
          "cloud.google.com/gke-os-distribution": "cos",
          "cloud.google.com/machine-family": "e2",
          "failure-domain.beta.kubernetes.io/region": "us-central1",
          "failure-domain.beta.kubernetes.io/zone": "us-central1-a",
          "iam.gke.io/gke-metadata-server-enabled": "true",
          "instance_labels_synced": "true",
          "k8s-nodepool-labeler": "13846344126847483082",
          "k8s_cluster": "qa-us-central-cluster",
          "kubernetes.io/arch": "amd64",
          "kubernetes.io/hostname": "gke-qa-us-central--auto-reply-pool-ad94ddd8-f132",
          "kubernetes.io/os": "linux",
          "node.kubernetes.io/instance-type": "e2-standard-2",
          "node.kubernetes.io/masq-agent-ds-ready": "true",
          "nodePool": "auto-reply-pool",
          "projectcalico.org/ds-ready": "true",
          "service": "auto-reply",
          "topology.gke.io/zone": "us-central1-a",
          "topology.kubernetes.io/region": "us-central1",
          "topology.kubernetes.io/zone": "us-central1-a"
        }
      },
      "timestamp": "2022-10-04T13:35:24Z",
      "window": "30s",
      "usage": {
        "cpu": "170334843n",
        "memory": "1636980Ki"
      }
    },
    ...


# Get the metrics for one nodes
kubectl get --raw /apis/metrics.k8s.io/v1beta1/nodes/NODE_NAME
```

We can see each node expose the `cpu` and `memory` metrics.

## list Pods metrics

```bash
kubectl get --raw "/apis/metrics.k8s.io/v1beta1/pods "|jq '.'

{
  "kind": "PodMetricsList",
  "apiVersion": "metrics.k8s.io/v1beta1",
  "metadata": {},
  "items": [
    {
    {
      "metadata": {
        "name": "auto-reply-584d9b45f5-6bhqx",
        "namespace": "auto-reply-v2",
        "creationTimestamp": "2022-10-04T13:37:22Z",
        "labels": {
          "app.kubernetes.io/instance": "auto-reply",
          "app.kubernetes.io/name": "auto-reply",
          "pod-template-hash": "584d9b45f5",
          "security.istio.io/tlsMode": "istio",
          "service.istio.io/canonical-name": "auto-reply",
          "service.istio.io/canonical-revision": "latest",
          "sidecar.istio.io/inject": "true",
          "topology.istio.io/network": "qa"
        }
      },
      "timestamp": "2022-10-04T13:36:51Z",
      "window": "30s",
      "containers": [
        {
          "name": "auto-reply",
          "usage": {
            "cpu": "109630n",
            "memory": "13436Ki"
          }
        },
        {
          "name": "istio-proxy",
          "usage": {
            "cpu": "11094142n",
            "memory": "110272Ki"
          }
        }
      ]
    },
    ...
    

# Get the metrics for one pods
kubectl get --raw /apis/metrics.k8s.io/v1beta1/namespaces/NAMESPACE/pods/POD_NAME
```
Here again we expose  `cpu` and `memory` metrics for each container in each pod.

## list cutom metrics

```bash
kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta1"
```

## list external metrics

```bash
kubectl get --raw "/apis/external.metrics.k8s.io/v1beta1"

```

# Pod management

## Delete all Terminated pods

```bash
kubectl get pods --all-namespaces | grep -i Terminated | awk '{print $1, $2}' | xargs -n2 kubectl delete pod -n
```
