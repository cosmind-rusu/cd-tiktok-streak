FROM golang:1.25-bookworm AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/cd-tiktok-streak .

FROM mcr.microsoft.com/playwright:v1.55.0-noble

WORKDIR /app

COPY --from=builder /out/cd-tiktok-streak /usr/local/bin/cd-tiktok-streak
COPY config.example.json /app/config.example.json
COPY deploy /app/deploy
RUN chmod +x /app/deploy/start.sh && mkdir -p /data

ENTRYPOINT ["/app/deploy/start.sh"]
