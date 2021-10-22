FROM golang:alpine AS builder

ARG TARGETOS=linux
ARG TARGETARCH

ADD . /litmus-go
WORKDIR /litmus-go

RUN export GOOS=${TARGETOS} && \
    export GOARCH=${TARGETARCH}

RUN CGO_ENABLED=0 go build -o /output/experiments ./bin/experiment


FROM alpine:3.13 AS dep

# Install generally useful things
RUN apk --update add \
        sudo \
        iproute2

# Packaging stage
# Image source: https://github.com/chaosnative/test-tools/blob/master/custom/hardened-alpine/experiment/Dockerfile
# The base image is non-root (have litmus user) with default litmus directory.
FROM chaosnative/experiment-alpine

LABEL maintainer="ChaosNative"

COPY --from=builder /output/ /litmus
COPY --from=dep /usr/bin/sudo /usr/bin/
COPY --from=dep /sbin/tc /sbin/