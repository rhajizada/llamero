FROM golang:1.25-alpine AS builder
WORKDIR /src
ARG VERSION

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build \
  -ldflags="-s -w -X main.Version=${VERSION}" \
  -o /src/bin/server \
  ./cmd/server

FROM alpine:3.18
RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /src/bin/server .
COPY --from=builder /src/data/sql/migrations data/sql/migrations

ENTRYPOINT ["./server"]
