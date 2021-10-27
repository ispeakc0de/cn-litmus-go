FROM golang:alpine AS builder

ARG TARGETOS=linux
ARG TARGETARCH

ADD . /litmus-go
WORKDIR /litmus-go

RUN export GOOS=${TARGETOS} && \
    export GOARCH=${TARGETARCH}

RUN apk add --update curl

#Installing govc cli binaries
RUN curl -L https://github.com/vmware/govmomi/releases/download/v0.26.1/govc_Linux_x86_64.tar.gz --output ~/govc_Linux_x86_64.tar.gz && mkdir ~/govc && tar xf ~/govc_Linux_x86_64.tar.gz -C ~/govc && cp ~/govc/govc /usr/local/bin && rm -r ~/govc && rm ~/govc_Linux_x86_64.tar.gz && chmod +x /usr/local/bin/govc

RUN CGO_ENABLED=0 go build -o /output/experiments ./bin/experiment

# Packaging stage
# Image source: https://github.com/litmuschaos/test-tools/blob/master/custom/hardened-alpine/experiment/Dockerfile
# The base image is non-root (have litmus user) with default litmus directory.
FROM litmuschaos/experiment-alpine

LABEL maintainer="ChaosNative"

COPY --from=builder /output/ /litmus