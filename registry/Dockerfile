FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY registry ./registry

COPY schemas ./schemas

WORKDIR /app/registry/registry_server

RUN go build -o /app/registry_server

CMD ["/app/registry_server"]