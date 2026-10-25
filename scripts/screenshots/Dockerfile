# Runs scripts/screenshots/run.ts; see docker-compose.screenshots.yml. `app` is the build context
# of the app's image (compose: additional_contexts). Build from the
# repository root. The image carries the browsers; the playwright package comes from
# scripts/package.json, and its version must be the one of the image tag below.
ARG PLAYWRIGHT_VERSION=1.63.0

# The throwaway OAuth app of the oauth-app scenario (cmd/oauth-probe, stdlib only), built here so
# the image needs no Go toolchain.
FROM golang:1.27-alpine AS probe
WORKDIR /src
COPY go.mod go.sum ./
COPY cmd/oauth-probe ./cmd/oauth-probe
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/oauth-probe ./cmd/oauth-probe

FROM mcr.microsoft.com/playwright:v${PLAYWRIGHT_VERSION}-noble
ARG PLAYWRIGHT_VERSION
WORKDIR /app/scripts
COPY scripts/package.json scripts/package-lock.json ./
RUN npm ci
RUN test "$(node -p "require('playwright/package.json').version")" = "${PLAYWRIGHT_VERSION}" \
    || { echo "scripts/package-lock.json has another Playwright than PLAYWRIGHT_VERSION=${PLAYWRIGHT_VERSION}: update both" >&2; exit 1; }
COPY scripts/screenshots ./screenshots
# Scenarios start the app themselves, each on an empty data directory (scenarios/server.ts).
COPY --from=app /usr/local/bin/savvy-go /usr/local/bin/savvy-go
COPY --from=probe /out/oauth-probe /usr/local/bin/oauth-probe
# The script reads the list of pages and the languages (a directory each) from the app's sources.
COPY resources/ts/app/pages.ts /app/resources/ts/app/pages.ts
COPY resources/ts/locales /app/resources/ts/locales
CMD ["npx", "tsx", "screenshots/run.ts"]
