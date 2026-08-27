FROM denoland/deno:2.9.1 AS frontend-builder
WORKDIR /app
COPY frontend/package.json frontend/deno.lock* ./
RUN deno install
COPY frontend/ .
RUN deno task build

FROM golang:1.27.0-alpine@sha256:4c9fe60190a2a3350ddc51de80d0224b8a6698d12bdfc999fee45ea9d6c46dbc AS builder
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY go.mod go.sum vendor ./
COPY . .
COPY --from=frontend-builder /app/dist frontend/dist
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /app/bin/app ./cmd/main/main.go

FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b
RUN apk add --no-cache ca-certificates wget
RUN addgroup -g 1001 -S appgroup
RUN adduser -u 1001 -S appuser -G appgroup

WORKDIR /app
COPY --chown=appuser:appgroup --from=builder /app/bin/app .
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1
CMD ["./app"]
