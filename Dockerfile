FROM alpine

RUN apk add --no-cache ca-certificates

COPY terramate-mcp-server /usr/local/bin/terramate-mcp-server
ENTRYPOINT ["/usr/local/bin/terramate-mcp-server"]
