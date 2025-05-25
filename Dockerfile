# Build stage
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

ARG VERSION=dev
ARG COMMIT_HASH
ARG BUILD_TIME

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X main.version=${VERSION} -s -w -extldflags '-static'" \
    -a -installsuffix cgo \
    -o mcp-stockfish .

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache \
    stockfish \
    ca-certificates \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1000 mcpuser && \
    adduser -D -s /bin/bash -u 1000 -G mcpuser mcpuser

# Copy binary from builder stage
COPY --from=builder /app/mcp-stockfish /usr/local/bin/mcp-stockfish
RUN chmod +x /usr/local/bin/mcp-stockfish

# Set environment
ENV MCP_STOCKFISH_PATH=/usr/bin/stockfish
ENV PATH="/usr/local/bin:${PATH}"

# Switch to non-root user
USER mcpuser
WORKDIR /home/mcpuser

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD echo '{"jsonrpc": "2.0", "method": "ping", "id": 1}' | timeout 2 mcp-stockfish || exit 1

ENTRYPOINT ["mcp-stockfish"]
