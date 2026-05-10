FROM golang:1.26.1-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/server ./cmd/server

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=docker:cli /usr/local/bin/docker /usr/local/bin/docker

RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    git \
    tar \
    && curl -sSL https://railpack.com/install.sh | sh \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/server /app/server

EXPOSE 8080

CMD ["/app/server"]
