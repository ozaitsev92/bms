# syntax=docker/dockerfile:1
FROM golang:1.25.8-alpine3.22 AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV CGO_ENABLED=0

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY models ./models

RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api && \
    GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/migrator ./cmd/migrator

FROM alpine:3.22

RUN addgroup -S app && adduser -S -G app -H app

WORKDIR /app

COPY --from=builder /out/api /out/migrator ./
COPY db/migrations ./migrations
COPY config ./config

ENV CONFIG_PATH=/app/config/local.yaml \
    MIGRATIONS_PATH=/app/migrations

USER app:app

EXPOSE 8080

CMD ["./api"]
