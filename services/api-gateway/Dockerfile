FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o api-gateway .


FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates wget

RUN addgroup -S bravogroup && \
    adduser -S bravouser -G bravogroup

COPY --from=builder /app/api-gateway .

USER bravouser

# The service listens this port
EXPOSE 8081

# Check whether the api-gateway is healthy.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -q --spider http://localhost:8081/healthz || exit 1

CMD ["./api-gateway"]