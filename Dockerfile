# syntax=docker/dockerfile:1.3-labs

FROM debian:bullseye-slim as base
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
RUN useradd -r -u 999 -d /home/cloud-provider-zero cloud-provider-zero

FROM ghcr.io/acorn-io/images-mirror/golang:1.21 AS build
COPY / /src
WORKDIR /src
RUN \
  --mount=type=cache,target=/go/pkg \
  --mount=type=cache,target=/root/.cache/go-build \
  go build -o bin/cloud-provider-zero main.go

FROM base AS goreleaser
COPY cloud-provider-zero /usr/local/bin/cloud-provider-zero
USER cloud-provider-zero

FROM base
COPY --from=build /src/bin/cloud-provider-zero /usr/local/bin/cloud-provider-zero
USER cloud-provider-zero
