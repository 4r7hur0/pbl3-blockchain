FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY api ./api 

COPY schemas ./schemas 

COPY registry ./registry

WORKDIR /app/api

RUN go build -o /app/api-server

FROM alpine:3.19

COPY --from=builder /app/api-server /app/api-server
CMD ["/app/api-server"]
