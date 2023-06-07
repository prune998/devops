FROM golang:1.20-alpine as builder
RUN apk add --no-cache ca-certificates git

ENV PROJECT github.com/prune998/devops/K8S/CPUconsumer
ARG VERSION="0.0.1-dev"

WORKDIR /go/src/$PROJECT

COPY . .
RUN GOOS=linux GOARCH=amd64 go install -v -ldflags="-X main.version=$VERSION" .

FROM alpine as release
RUN apk add --no-cache ca-certificates \
    busybox-extras net-tools bind-tools curl
WORKDIR /
COPY --from=builder /go/bin/linux_amd64/CPUconsumer /CPUconsumer

ENTRYPOINT ["/CPUconsumer"]
