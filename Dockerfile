# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Build arguments for version information
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary for the target architecture
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT} -X main.BuildTime=${BUILD_TIME}" \
    -trimpath \
    -o terramate-mcp-server \
    ./cmd/terramate-mcp-server

# Runtime stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates

COPY --from=builder /build/terramate-mcp-server /usr/local/bin/terramate-mcp-server

# The server supports two authentication methods:
#
# 1. JWT Token (Recommended):
#    Mount your credential directory and provide region:
#    docker run -v ~/.terramate.d:/root/.terramate.d:ro \
#               -e TERRAMATE_REGION=eu \
#               ghcr.io/terramate-io/terramate-mcp-server
#
# 2. API Key (Alternative approach, creating and managing API keys requires admin privileges):
#    Provide API key and region via environment variables:
#    docker run -e TERRAMATE_API_KEY=your-key \
#               -e TERRAMATE_REGION=eu \
#               ghcr.io/terramate-io/terramate-mcp-server
#
# For more information: https://github.com/terramate-io/terramate-mcp-server

ENTRYPOINT ["/usr/local/bin/terramate-mcp-server"]
