# Build stage 
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o inventory-service .

# Runtime stage
FROM alpine:3.22

WORKDIR /app

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy only the compiled binary
COPY --from=builder /app/inventory-service /app/inventory-service

EXPOSE 8082

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8082/healthz || exit 1

USER appuser

CMD ["/app/inventory-service"]