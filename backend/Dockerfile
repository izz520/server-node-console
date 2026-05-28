FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api

FROM alpine:3.22

ARG TARGETARCH
ARG MIHOMO_VERSION=v1.19.22

RUN apk add --no-cache ca-certificates tzdata wget gzip \
  && case "$TARGETARCH" in \
    amd64) MIHOMO_ARCH=amd64-compatible ;; \
    arm64) MIHOMO_ARCH=arm64 ;; \
    *) echo "unsupported TARGETARCH: $TARGETARCH" >&2; exit 1 ;; \
  esac \
  && wget -O /usr/local/bin/mihomo.gz "https://github.com/MetaCubeX/mihomo/releases/download/${MIHOMO_VERSION}/mihomo-linux-${MIHOMO_ARCH}-${MIHOMO_VERSION}.gz" \
  && gzip -d /usr/local/bin/mihomo.gz \
  && chmod +x /usr/local/bin/mihomo

WORKDIR /app
COPY --from=builder /out/api /app/api

EXPOSE 8080
CMD ["/app/api"]
