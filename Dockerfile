FROM golang:1.25-alpine AS builder

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY internal ./internal
COPY cmd ./cmd
COPY models ./models

RUN go build -o /app/api ./cmd/api
RUN go build -o /app/migrator ./cmd/migrator

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/api ./api
COPY --from=builder /app/migrator ./migrator
COPY db/migrations ./migrations
COPY config ./config

EXPOSE 8080

CMD ["./api"]