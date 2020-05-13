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