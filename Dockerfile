FROM docker.io/library/golang:1.25-alpine AS builder

RUN addgroup -S appgroup && adduser -S appuser -G appgroup \
    && mkdir /prod-dir \
    && chown appuser:appgroup /prod-dir \
    && apk add --no-cache \
        ca-certificates=20251003-r0 \
        tzdata=2026a-r0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build \
        -ldflags="-s -w" \
        -trimpath \
        -o /wishlist-service \
        ./cmd/wishlist/main.go

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo                 /usr/share/zoneinfo
COPY --from=builder /etc/passwd                         /etc/passwd
COPY --from=builder /etc/group                          /etc/group

COPY --from=builder --chown=appuser:appgroup /prod-dir  /app

WORKDIR /app

COPY --from=builder --chown=appuser:appgroup /app/migrations ./migrations

COPY --from=builder \
     --chown=appuser:appgroup \
     --chmod=755 \
     /wishlist-service ./wishlist-service

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/wishlist-service"]