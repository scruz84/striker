# syntax=docker/dockerfile:1

## Build
FROM golang:1.21.5-alpine AS build

RUN apk --no-cache add ca-certificates

WORKDIR /app/

COPY fmcommon/ /app/fmcommon/
COPY ../backend/ /app/striker-backend/

WORKDIR /app/striker-backend/cmd
RUN go mod download

RUN CGO_ENABLED=0 go build -o /striker

## Deploy
FROM debian:stable

RUN apt-get update && apt-get install -y ca-certificates openssl
RUN update-ca-certificates

WORKDIR /app
COPY --from=build /striker /app/striker/
EXPOSE 1984

ENTRYPOINT ["/app/striker"]