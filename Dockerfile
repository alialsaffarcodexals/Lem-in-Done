# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app
RUN apk add --no-cache build-base
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /out/forum ./cmd/server

# Runtime
FROM alpine:3.20
WORKDIR /srv
RUN mkdir -p /srv/data
# run as root for simplest perms
COPY --from=builder /out/forum /srv/forum
COPY internal /srv/internal
EXPOSE 8080
ENTRYPOINT ["/srv/forum","-addr=:8080","-data=/srv/data","-templates=/srv/internal/web/templates"]
