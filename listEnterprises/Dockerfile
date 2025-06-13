FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

WORKDIR /app/listEnterprises

RUN go build -o /app/lisCarMqtt .

CMD ["/app/lisCarMqtt"]