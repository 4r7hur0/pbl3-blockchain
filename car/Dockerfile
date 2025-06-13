FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY car ./car 

COPY schemas ./schemas

WORKDIR /app/car

RUN go build -o /app/car-app .

FROM alpine:3.19

WORKDIR /app
COPY --from=builder /app/car-app .

CMD ["./car-app"]