# syntax=docker/dockerfile:1
#
# Build context for this Dockerfile is the backend/ directory (see docker-compose.yml).

## Build
FROM golang:1.27-alpine AS build

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /striker ./cmd

## Deploy
FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates openssl \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates

WORKDIR /app
COPY --from=build /striker /app/striker

EXPOSE 1984

ENTRYPOINT ["/app/striker"]
