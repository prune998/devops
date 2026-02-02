FROM golang:1.25-alpine AS builder
RUN apk add --no-cache ca-certificates git

ENV PROJECT=github.com/prune998/devops/CLOUD/GCP/pubsub-publish
ARG VERSION="0.0.1-dev"

WORKDIR /go/src/$PROJECT

COPY . .
RUN go get ./...
RUN GOOS=linux GOARCH=amd64 go install -v -ldflags="-X main.version=$VERSION" .

FROM alpine AS release
# ENV GCP_PROJECT_ID=my-project
# ENV PUBSUB_TOPIC_ID=test

RUN apk add --no-cache ca-certificates \
    busybox-extras net-tools bind-tools curl
WORKDIR /
COPY --from=builder /go/bin/linux_amd64/pubsub-publish /pubsub-publish

ENTRYPOINT ["/pubsub-publish"]
