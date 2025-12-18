# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /api ./cmd/api

# ── Final stage ───────────────────────────────────────────────────────────────
FROM alpine:3.20

# ca-certificates needed for outbound HTTPS (TMDB, Open Library, Google Books)
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /api .
COPY migrations/ migrations/

EXPOSE 8000

ENTRYPOINT ["/app/api"]
