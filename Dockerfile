# ---- build stage ----
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod* go.sum* ./
# Initialise a module if go.mod is not committed yet (no external dependencies)
RUN [ -f go.mod ] || go mod init webhook-translator
RUN go mod download 2>/dev/null || true
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o webhook-translator .

# ---- runtime stage ----
FROM alpine:3.21

RUN apk add --no-cache ca-certificates

RUN addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=builder /app/webhook-translator .

USER app
EXPOSE 80
ENTRYPOINT ["/app/webhook-translator"]
