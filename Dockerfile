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
FROM 343218184206.dkr.ecr.us-east-1.amazonaws.com/proxy/library/golang:1.24-alpine AS builder

WORKDIR /usr/src/mcp
COPY --chown=root:root . /usr/src/mcp

ARG BUILD_VERSION=dev
ARG GOPRIVATE=github.com/teamwork

RUN apk add openssh-client git

# Force fetching modules over SSH
RUN git config --system url."ssh://git@github.com/".insteadOf "https://github.com/"

# Fetch github's SSH host keys and compare them to the published
# ones at:
#
# https://help.github.com/en/articles/githubs-ssh-key-fingerprints
RUN set -eu && \
    mkdir -p -m 0600 ~/.ssh && \
    ssh-keyscan -t rsa github.com >> ~/.ssh/known_hosts && \
    ssh-keygen -F github.com -l -E sha256 \
        | grep -q "SHA256:uNiVztksCsDhcc0u9e8BujQXVUpKZIDTMczCvj3tD2s"

RUN --mount=type=ssh go mod download
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
FROM 343218184206.dkr.ecr.us-east-1.amazonaws.com/proxy/library/alpine:3 AS runner

COPY --from=builder /app/tw-mcp-http /bin/tw-mcp-http
COPY --from=builder /app/tw-mcp-stdio /bin/tw-mcp-stdio

ARG BUILD_DATE
ARG BUILD_VCS_REF
ARG BUILD_VERSION

ENV TW_MCP_VERSION $BUILD_VERSION

LABEL org.label-schema.build-date=$BUILD_DATE \
      org.label-schema.description="Teamwork MCP server" \
      org.label-schema.name="mcp" \
      org.label-schema.schema-version="1.0" \
      org.label-schema.url="https://github.com/teamwork/mcp" \
      org.label-schema.vcs-url="https://github.com/teamwork/mcp" \
      org.label-schema.vcs-ref=$BUILD_VCS_REF \
      org.label-schema.vendor="Teamwork" \
      org.label-schema.version=$BUILD_VERSION

ENTRYPOINT ["/bin/tw-mcp-http"]