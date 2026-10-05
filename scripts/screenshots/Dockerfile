# Runs scripts/screenshots/run.ts; see docker-compose.screenshots.yml. Build from the
# repository root. The image carries the browsers; the playwright package comes from
# scripts/package.json, and its version must be the one of the image tag below.
ARG PLAYWRIGHT_VERSION=1.63.0
FROM mcr.microsoft.com/playwright:v${PLAYWRIGHT_VERSION}-noble
ARG PLAYWRIGHT_VERSION
WORKDIR /app/scripts
COPY scripts/package.json scripts/package-lock.json ./
RUN npm ci
RUN test "$(node -p "require('playwright/package.json').version")" = "${PLAYWRIGHT_VERSION}" \
    || { echo "scripts/package-lock.json has another Playwright than PLAYWRIGHT_VERSION=${PLAYWRIGHT_VERSION}: update both" >&2; exit 1; }
COPY scripts/screenshots ./screenshots
# The script reads the list of pages from the app's sources.
COPY resources/ts/app/pages.ts /app/resources/ts/app/pages.ts
CMD ["npx", "tsx", "screenshots/run.ts"]
