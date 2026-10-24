# syntax=docker/dockerfile:1
# Build the frontend and binary from this checkout; no published image is required.
FROM --platform=$BUILDPLATFORM node:22-bookworm-slim AS frontend
WORKDIR /src/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build:static

FROM --platform=$BUILDPLATFORM golang:1.26.10-bookworm AS backend
ARG TARGETOS=linux
ARG TARGETARCH
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /src/web/out/ ./server/webui/dist/
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -tags embedui -trimpath \
    -ldflags "-s -w -X main.version=${VERSION}" -o /blacksand ./cmd/blacksand

FROM python:3.12-slim-bookworm
# Runtime tools and Node 22 for Playwright and the E2B bridge.
RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates ripgrep curl wget vim git jq unzip \
      dnsutils iputils-ping netcat-openbsd inetutils-telnet whois nmap \
    && curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && rm -rf /var/lib/apt/lists/*
RUN npm install -g @playwright/mcp@latest @playwright/cli@latest playwright@latest \
    && playwright-cli --help \
    && playwright install --with-deps chromium \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=backend /blacksand /app/blacksand
# The supervisor forwards SIGTERM and restarts after a self-update.
COPY start.sh /app/start.sh
RUN chmod +x /app/blacksand /app/start.sh
COPY skills/ /app/skills/
COPY adapters/e2b/ /app/adapters/e2b/
RUN cd /app/adapters/e2b && npm ci --omit=dev --ignore-scripts
COPY LICENSE /app/LICENSE
VOLUME ["/app/data"]
EXPOSE 8787 8788
ENTRYPOINT ["/app/start.sh"]
CMD ["-addr", ":8787", "-proxy", ":8788"]
