# syntax=docker/dockerfile:1

# ▄▄▄▄    █    ██  ██▓ ██▓    ▓█████▄ ▓█████  ██▀███  
# ▓█████▄  ██  ▓██▒▓██▒▓██▒    ▒██▀ ██▌▓█   ▀ ▓██ ▒ ██▒
# ▒██▒ ▄██▓██  ▒██░▒██▒▒██░    ░██   █▌▒███   ▓██ ░▄█ ▒
# ▒██░█▀  ▓▓█  ░██░░██░▒██░    ░▓█▄   ▌▒▓█  ▄ ▒██▀▀█▄  
# ░▓█  ▀█▓▒▒█████▓ ░██░░██████▒░▒████▓ ░▒████▒░██▓ ▒██▒
# ░▒▓███▀▒░▒▓▒ ▒ ▒ ░▓  ░ ▒░▓  ░ ▒▒▓  ▒ ░░ ▒░ ░░ ▒▓ ░▒▓░
# ▒░▒   ░ ░░▒░ ░ ░  ▒ ░░ ░ ▒  ░ ░ ▒  ▒  ░ ░  ░  ░▒ ░ ▒░
#  ░    ░  ░░░ ░ ░  ▒ ░  ░ ░    ░ ░  ░    ░     ░░   ░ 
#  ░         ░      ░      ░  ░   ░       ░  ░   ░     
#       ░                       ░                      
#
FROM golang:1.25-alpine AS builder

WORKDIR /usr/src/mcp
COPY --chown=root:root . /usr/src/mcp

ARG BUILD_VERSION=dev

RUN go mod download
RUN go build -ldflags="-X 'github.com/teamwork/mcp/internal/config.Version=$BUILD_VERSION'" -o /app/tw-mcp-http ./cmd/mcp-http
RUN go build -ldflags="-X 'github.com/teamwork/mcp/internal/config.Version=$BUILD_VERSION'" -o /app/tw-mcp-stdio ./cmd/mcp-stdio


# ██▀███   █    ██  ███▄    █  ███▄    █ ▓█████  ██▀███  
# ▓██ ▒ ██▒ ██  ▓██▒ ██ ▀█   █  ██ ▀█   █ ▓█   ▀ ▓██ ▒ ██▒
# ▓██ ░▄█ ▒▓██  ▒██░▓██  ▀█ ██▒▓██  ▀█ ██▒▒███   ▓██ ░▄█ ▒
# ▒██▀▀█▄  ▓▓█  ░██░▓██▒  ▐▌██▒▓██▒  ▐▌██▒▒▓█  ▄ ▒██▀▀█▄  
# ░██▓ ▒██▒▒▒█████▓ ▒██░   ▓██░▒██░   ▓██░░▒████▒░██▓ ▒██▒
# ░ ▒▓ ░▒▓░░▒▓▒ ▒ ▒ ░ ▒░   ▒ ▒ ░ ▒░   ▒ ▒ ░░ ▒░ ░░ ▒▓ ░▒▓░
#   ░▒ ░ ▒░░░▒░ ░ ░ ░ ░░   ░ ▒░░ ░░   ░ ▒░ ░ ░  ░  ░▒ ░ ▒░
#   ░░   ░  ░░░ ░ ░    ░   ░ ░    ░   ░ ░    ░     ░░   ░ 
#    ░        ░              ░          ░    ░  ░   ░     
#
FROM alpine:3 AS runner

COPY --from=builder /app/tw-mcp-http /bin/tw-mcp-http
COPY --from=builder /app/tw-mcp-stdio /bin/tw-mcp-stdio

# Set permissions for the binaries
RUN addgroup -S mcp && adduser -S mcp -G mcp
RUN chown mcp:mcp /bin/tw-mcp-http /bin/tw-mcp-stdio && chmod 0755 /bin/tw-mcp-http /bin/tw-mcp-stdio
USER mcp

ARG BUILD_DATE
ARG BUILD_VCS_REF
ARG BUILD_VERSION

ENV TW_MCP_VERSION=$BUILD_VERSION

LABEL org.label-schema.build-date=$BUILD_DATE \
      org.label-schema.description="Teamwork.com MCP server" \
      org.label-schema.name="mcp" \
      org.label-schema.schema-version="1.0" \
      org.label-schema.url="https://github.com/teamwork/mcp" \
      org.label-schema.vcs-url="https://github.com/teamwork/mcp" \
      org.label-schema.vcs-ref=$BUILD_VCS_REF \
      org.label-schema.vendor="Teamwork.com" \
      org.label-schema.version=$BUILD_VERSION \
      io.modelcontextprotocol.server.name="com.teamwork/mcp"

ENTRYPOINT [ "/bin/tw-mcp-http" ]

#   ██████ ▄▄▄█████▓▓█████▄  ██▓ ▒█████  
# ▒██    ▒ ▓  ██▒ ▓▒▒██▀ ██▌▓██▒▒██▒  ██▒
# ░ ▓██▄   ▒ ▓██░ ▒░░██   █▌▒██▒▒██░  ██▒
#   ▒   ██▒░ ▓██▓ ░ ░▓█▄   ▌░██░▒██   ██░
# ▒██████▒▒  ▒██▒ ░ ░▒████▓ ░██░░ ████▓▒░
# ▒ ▒▓▒ ▒ ░  ▒ ░░    ▒▒▓  ▒ ░▓  ░ ▒░▒░▒░ 
# ░ ░▒  ░ ░    ░     ░ ▒  ▒  ▒ ░  ░ ▒ ▒░ 
# ░  ░  ░    ░       ░ ░  ░  ▒ ░░ ░ ░ ▒  
#       ░              ░     ░      ░ ░  
#                    ░                   
FROM runner AS stdio

ENTRYPOINT [ "/bin/tw-mcp-stdio" ]