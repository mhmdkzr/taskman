FROM denoland/deno:2.9.1 AS frontend-builder
WORKDIR /app
COPY frontend/package.json frontend/deno.lock* ./
RUN deno install
COPY frontend/ .
RUN deno task build

FROM golang:1.26.4-alpine@sha256:f1ddd9fe14fffc091dd98cb4bfa999f32c5fc77d2f2305ea9f0e2595c5437c14 AS builder
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
