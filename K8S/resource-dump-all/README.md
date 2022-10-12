# go-k8s-list-all-object
List kubernetes object in the current context

# How to
## List all namespaced object in all namespace
```
go run .
```

## List all namespaced object and include non-namespaced object
```
go run . -a
```

## List all object in all namespace based on labels
```
go run . -l "app.kubernetes.io/name=core-api"
```

## List all object in a specific namespace
```
go run . -n core-api-v2
```

## List all object in a specific namespace based on labels
```
go run . -n core-api-v2 -l "app.kubernetes.io/name=core-api"
```
