FROM golang:1.14-alpine as builder
RUN apk add --no-cache ca-certificates git

ENV PROJECT github.com/prune998/devops/WEB/droppingServer
ARG VERSION="0.0.1-dev"

WORKDIR /go/src/$PROJECT

COPY . .
RUN go install -v .

FROM alpine as release
RUN apk add --no-cache ca-certificates \
    busybox-extras net-tools bind-tools curl
WORKDIR /
COPY --from=builder /go/bin/droppingServer /droppingServer
COPY server.crt /server.crt
COPY server.key /server.key
EXPOSE 8080
ENTRYPOINT ["/droppingServer"]
