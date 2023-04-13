# syntax=docker/dockerfile:1

## Build
FROM golang:1.20.2-alpine AS build

RUN apk --no-cache add ca-certificates

WORKDIR /app/

COPY fmcommon/ /app/fmcommon/
COPY fmserver/ /app/fmserver/

WORKDIR /app/fmcommon
RUN go mod download

WORKDIR /app/fmserver
RUN go mod download

RUN CGO_ENABLED=0 go build -o /fmserver

## Deploy
FROM debian:stable

RUN apt-get update && apt-get install -y ca-certificates openssl
RUN update-ca-certificates

WORKDIR /app
COPY --from=build /fmserver /app/fmserver
EXPOSE 1984

ENTRYPOINT ["/app/fmserver"]