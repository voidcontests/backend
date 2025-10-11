FROM golang:1.25.1-alpine3.22 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o build/api ./cmd/api

# Lightweight docker container with binaries only
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/build ./build

CMD ["./build/api"]
