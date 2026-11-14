# The story tests in Chromium, for `npm run stories` (scripts/stories/main.mts, which runs itself in it
# with --local). Build from the repository root. Nothing is mounted: the sources are copied in, so any
# Docker daemon works. The image carries the browsers; the playwright package comes from the root
# package-lock.json, and its version must be the one of the image tag below.
ARG PLAYWRIGHT_VERSION=1.64.1

FROM mcr.microsoft.com/playwright:v${PLAYWRIGHT_VERSION}-noble
ARG PLAYWRIGHT_VERSION
WORKDIR /app
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
RUN test "$(node -p "require('playwright/package.json').version")" = "${PLAYWRIGHT_VERSION}" \
    || { echo "package-lock.json has another Playwright than PLAYWRIGHT_VERSION=${PLAYWRIGHT_VERSION}: update both" >&2; exit 1; }
COPY tsconfig.json vite.config.ts vitest.stories.config.ts components.json ./
# Merged into .storybook, so the service worker `npm ci` wrote there (prepare: msw init) stays.
COPY .storybook ./.storybook
COPY resources ./resources
COPY scripts/runner.mts ./scripts/
COPY scripts/stories ./scripts/stories
CMD ["node", "scripts/stories/main.mts", "--local"]
