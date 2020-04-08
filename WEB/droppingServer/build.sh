#!/bin/bash

set -x

VERSION=${VERSION:-v0.0.1}
docker build --build-arg VERSION=${VERSION} --tag  prune/droppingserver:${VERSION}  .
docker push prune/droppingserver:${VERSION}