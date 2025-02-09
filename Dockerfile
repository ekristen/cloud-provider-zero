# syntax=docker/dockerfile:1.13-labs
FROM cgr.dev/chainguard/wolfi-base:latest as base
ARG PROJECT_NAME=cloud-provider-zero
RUN apk add --no-cache ca-certificates
RUN addgroup -S ${PROJECT_NAME} && adduser -S ${PROJECT_NAME} -G ${PROJECT_NAME}

FROM ghcr.io/acorn-io/images-mirror/golang:1.21 AS build
ARG PROJECT_NAME=cloud-provider-zero
COPY / /src
WORKDIR /src
RUN \
  --mount=type=cache,target=/go/pkg \
  --mount=type=cache,target=/root/.cache/go-build \
  go build -o bin/${PROJECT_NAME} main.go

FROM base AS goreleaser
ARG PROJECT_NAME=cloud-provider-zero
COPY ${PROJECT_NAME} /usr/local/bin/${PROJECT_NAME}
ENTRYPOINT ["/usr/local/bin/cloud-provider-zero"]
USER ${PROJECT_NAME}

FROM base
ARG PROJECT_NAME=cloud-provider-zero
COPY --from=build /src/bin/${PROJECT_NAME} /usr/local/bin/${PROJECT_NAME}
ENTRYPOINT ["/usr/local/bin/cloud-provider-zero"]
USER ${PROJECT_NAME}