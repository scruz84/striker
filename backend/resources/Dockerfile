# syntax=docker/dockerfile:1
#
# Build context for this Dockerfile is the backend/ directory (see the root docker-compose.yml).
#
# Multi-platform (linux/amd64 and linux/arm64, see .github/workflows/docker.yml): the build stage
# always runs on the builder's own platform and cross-compiles for the target one, which is much
# faster than compiling under emulation. Only the small runtime stage is per platform.

## Build
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
ARG TARGETOS
ARG TARGETARCH

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /striker ./cmd

## Deploy
FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates openssl \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates

WORKDIR /app
COPY --from=build /striker /app/striker

EXPOSE 1984

ENTRYPOINT ["/app/striker"]
