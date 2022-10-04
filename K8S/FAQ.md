# FAQ for Kubernetes

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

# Istio

## Limit config sent to a Gateway

set `PILOT_FILTER_GATEWAY_CLUSTER_CONFIG` to `true`

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

## list cutom metrics

```bash
kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta1"
```

## list external metrics

```bash
kubectl get --raw "/apis/external.metrics.k8s.io/v1beta1"

```
