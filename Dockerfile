# Build context is the repository root, because the backend builds against the
# shared ./pkg-common module through go.work.
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.4

WORKDIR /app

# Dependencies first, so they cache.
COPY go.work go.work.sum* ./
COPY backend/go.mod backend/go.sum ./backend/
COPY pkg-common/go.mod pkg-common/go.sum ./pkg-common/
RUN cd backend && go mod download

COPY backend/ backend/
COPY pkg-common/ pkg-common/

WORKDIR /app/backend
RUN swag init -g cmd/api/main.go -o docs
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/server ./cmd/api

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget \
    && adduser -D -u 10001 appuser

WORKDIR /app
COPY --from=builder /app/server .
USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/healthcheck || exit 1

CMD ["./server"]
