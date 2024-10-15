# Devops

This is a bunch of scripts/tools I used at some locations. Some are from me, some are simple copy of someone else's work tuned for my specific needs. 

You will find more docs inside each folder

## Update Go dependencies

```bash
# update the go version in all those files
for i in $(find . -type f -name go.mod) ; do echo $i ; done

# update deps
for i in $(find . -type f -name go.mod) ; do DIR=$(dirname $i) ; echo $DIR ; pushd  $DIR ; go get -u ./... ; popd ; done
```

## SSL

SSL related scripts, like generating full-feature Self-Signed certs

## DNS

DNS relared tools, like to use the [Gandi](https://gandi.net) API as a dynamic DNS

