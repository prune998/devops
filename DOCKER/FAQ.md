# Docker FAQ

## Build a Go app

### using mount-bind

see https://github.com/moby/buildkit/blob/master/frontend/dockerfile/docs/reference.md#run---mounttypebind

```
FROM golang:1.23 AS builder
ENV CGO_ENABLED="0"
ENV GOAMD64=v3

WORKDIR /go/src/app

ARG CI_CONCURRENT_PROJECT_ID="0"
ARG CI_COMMIT_SHORT_SHA=""
ARG CI_COMMIT_REF_NAME="next"

RUN --mount=type=bind go build -ldflags="-s -w \
    -X main.Version=${CI_COMMIT_REF_NAME} \
    -X main.Build=${CI_CONCURRENT_PROJECT_ID} \
    -X main.GitHash=${CI_COMMIT_SHORT_SHA} \
    " -o /builds/
RUN go version -m /builds/*

# Main runtime container
FROM gcr.io/distroless/static
COPY --from=builder /builds/k8s-graceful-shutdown-helper /bin/

USER 65534:65534
ENTRYPOINT ["/bin/k8s-graceful-shutdown-helper"]
```