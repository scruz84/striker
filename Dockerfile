# syntax=docker/dockerfile:1

## Build
FROM golang:1.20.2-alpine AS build

WORKDIR /app/

COPY fmcommon/ /app/fmcommon/
COPY fmserver/ /app/fmserver/

WORKDIR /app/fmcommon
RUN go mod download

WORKDIR /app/fmserver
RUN go mod download

RUN go build -o /fmserver

## Deploy
FROM debian:stable-slim

WORKDIR /app
COPY --from=build /fmserver /app/fmserver
EXPOSE 1984

ENTRYPOINT ["/app/fmserver"]