FROM golang:1.23-alpine3.20 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o .bin/server cmd/server/main.go

# Lightweight docker container with binaries only
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/.bin ./bin

CMD [ "./bin/server" ]
