FROM golang:1.25.1-alpine3.22 AS builder

WORKDIR /app

RUN apk add --no-cache git # purely for baking commit and branch into executable

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -a -ldflags "-w -s \
    -X github.com/voidcontests/api/internal/version.Commit=$(git rev-parse --short HEAD) \
    -X github.com/voidcontests/api/internal/version.Branch=$(git rev-parse --abbrev-ref HEAD)" \
    -o build/api ./cmd/api

# lightweight docker container with binaries only
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/build ./build

CMD ["./build/api"]
