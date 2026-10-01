# Scripts

## `npm run logos`

Regenerates every static logo file from the `Logo` React component in [`resources/ts/components/shared/Logo.tsx`](../resources/ts/components/shared/Logo.tsx), which is the single source of truth for the logo. The script ([`scripts/update-logos.ts`](../scripts/update-logos.ts)) renders the component to SVG and writes `public/favicon.svg` (with a dark-theme colour), the PNG icons (`favicon-96x96`, `web-app-manifest-192x192`/`512x512`, `apple-touch-icon`) and a multi-size `favicon.ico` into `public/`, plus the light and dark README logos into `docs/images/`. Edit the component, then run the command. The scripts have their own [`package.json`](../scripts/package.json) (`tsx`, `sharp`), so they never end up in the app's dependencies or the Docker build; install them once with `npm --prefix scripts install`. The script also imports `react`, so the root `npm install` is needed too.
