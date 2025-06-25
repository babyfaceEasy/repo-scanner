FROM golang:1.20 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /repo-scanner ./cmd/repo-scanner

FROM alpine:3.20

COPY --from=builder /repo-scanner /usr/local/bin/
COPY .env /app/.env

WORKDIR /app

ENTRYPOINT ["/usr/local/bin/repo-scanner"]