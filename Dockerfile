# syntax=docker/dockerfile:1
#
#  CI  binaries job   ( Go, QEMU ),  
#   dist/<TARGETARCH>/artex  .      arm64  apt 
#
#   cd web && npm run build:static && cd ..
#   mkdir -p server/webui && cp -r web/out server/webui/dist
#   CGO_ENABLED=0 GOARCH=amd64 go build -tags embedui -o dist/amd64/artex ./cmd/artex
#   docker build -t artex:local .
FROM python:3.12-slim-bookworm
ARG TARGETARCH
#  : ripgrep / curl / vim   recon     (  ·).
# Node  NodeSource  20.x  : bookworm  apt nodejs  18 , Playwright  >=20  .
RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates ripgrep curl wget vim git jq unzip \
      dnsutils iputils-ping netcat-openbsd inetutils-telnet whois nmap \
    && curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && rm -rf /var/lib/apt/lists/*
# Playwright MCP  CLI    ( npx    ).
# @playwright/mcp: browser MCP  `npx @playwright/mcp`   (  , -y/@latest ).
# @playwright/cli: playwright-cli  ,   --help     .
#  playwright(  ) , --with-deps  chromium       .
RUN npm install -g @playwright/mcp@latest @playwright/cli@latest playwright@latest \
    && playwright-cli --help \
    && playwright install --with-deps chromium \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
#     (dist/amd64/artex  dist/arm64/artex)
COPY dist/${TARGETARCH}/artex /app/artex
#    SIGTERM  artex    : docker stop   PID 1  ,
#   artex         10   SIGKILL   .
COPY start.sh /app/start.sh
RUN chmod +x /app/artex /app/start.sh
COPY skills/ /app/skills/
# data/(SQLite + jwt.key)  
VOLUME ["/app/data"]
EXPOSE 8787 8788
ENTRYPOINT ["/app/start.sh"]
CMD ["-addr", ":8787", "-proxy", ":8788"]
