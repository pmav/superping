# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
	go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/superping ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates iputils

WORKDIR /app
COPY --from=build /out/superping /app/superping

RUN mkdir /data
VOLUME ["/data"]
ENV SUPERPING_ADDRESS=:8080
ENV SUPERPING_DATABASE=/data/superping.db

EXPOSE 8080
ENTRYPOINT ["/app/superping"]