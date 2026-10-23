/// <reference types="vitest/config" />
import fs from 'fs'
import path from 'path'
import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { storybookTest } from '@storybook/addon-vitest/vitest-plugin'
import { playwright } from '@vitest/browser-playwright'
import pkg from './package.json'
import { DESKTOP_ONLY, MOBILE_ONLY } from './.storybook/tags'

// Every story is a test, in a real browser at one size (`npm run test:stories` runs both projects, English and light).
// A story tagged for the other size is left out of this project; see .storybook/tags.ts.
// `npm run test:stories:full` adds the language and theme matrix: the same stories again with another global `locale`
// or `theme` (see .storybook/preview.tsx and docs/development.md).

// Every locale folder except the English source is one more matrix project, so a new language needs no change here.
const EXTRA_LOCALES = fs
    .readdirSync(path.join(__dirname, 'resources/ts/locales'), { withFileTypes: true })
    .filter((entry) => entry.isDirectory() && entry.name !== 'en')
    .map((entry) => entry.name)

// A play function finds its elements by English text, so in another language it would only fail. The stories of such a
// project are rendered and checked (.storybook/matrix-checks.ts) without it; the English run covers what play does.
const withoutPlay = () => ({
    name: 'stories-without-play',
    enforce: 'post' as const,
    transform(code: string, id: string) {
        if (!/\.stories\.tsx?$/.test(id.split('?')[0])) return null
        const objects = [...code.matchAll(/^export (?:const|default) (\w+)/gm)].map((match) => match[1])
        if (objects.length === 0) return null
        const clear = `for (const story of [${objects.join(', ')}]) if (story && typeof story === 'object') story.play = undefined`
        return { code: `${code}\n;${clear}\n`, map: null }
    },
})

const storiesProject = (
    name: string,
    viewport: { width: number; height: number },
    excludeTags: string[],
    globals: { locale?: string; theme?: string } = {},
) => ({
    extends: true as const,
    plugins: [
        ...(globals.locale ? [withoutPlay()] : []),
        storybookTest({ configDir: path.join(__dirname, '.storybook'), tags: { exclude: excludeTags } }),
    ],
    // Name every dependency up front; one found mid-run makes Vite reload and fail the run.
    optimizeDeps: {
        include: [
            ...Object.keys(pkg.dependencies).filter((dep) => !dep.startsWith('@vitejs/')),
            // Deep imports are not found from the package name.
            'nuqs/adapters/testing',
            'echarts/core',
            'echarts/charts',
            'echarts/components',
            'echarts/renderers',
            'echarts-for-react/lib/core',
        ],
    },
    test: {
        name,
        // Read by initialGlobals in .storybook/preview.tsx; unset means English and light. The checks in
        // .storybook/matrix-checks.ts stay off in the Storybook UI.
        env: {
            STORYBOOK_LOCALE: globals.locale ?? 'en',
            STORYBOOK_THEME: globals.theme ?? 'light',
            // addon-vitest sizes the browser by the story's viewport global before every story and ignores the size set
            // below, so the preview picks the matching option (mobile and desktop in .storybook/preview.tsx).
            STORYBOOK_VIEWPORT: viewport.width < 600 ? 'mobile' : 'desktop',
            STORYBOOK_MATRIX_CHECKS: 'on',
            STORYBOOK_CONTRAST: globals.theme === 'dark' ? (process.env.STORYBOOK_CONTRAST ?? '') : '',
        },
        // Many stories load at once on a cold cache; the Russian ones wait for their language files.
        testTimeout: 30_000,
        hookTimeout: 30_000,
        browser: {
            enabled: true,
            headless: true,
            provider: playwright(),
            instances: [{ browser: 'chromium' as const, viewport }],
        },
    },
})

export default defineConfig(({ mode }) => {
    const env = loadEnv(mode, process.cwd(), '')

    return {
        plugins: [react(), tailwindcss()],
        base: '/build/',
        publicDir: false,
        resolve: {
            alias: {
                '@': path.resolve(__dirname, './resources/ts'),
            },
        },
        define: {
            __APP_VERSION__: JSON.stringify(env.APP_VERSION || process.env.APP_VERSION || 'dev'),
            // Test ids (lib/test-id.ts), for scripts/screenshots; left out of every other build.
            __TEST_IDS__: JSON.stringify((env.APP_ENV || process.env.APP_ENV) === 'screenshots'),
        },
        build: {
            outDir: 'public/build',
            emptyOutDir: true,
            manifest: true,
            chunkSizeWarningLimit: 800,
            rollupOptions: {
                input: path.resolve(__dirname, 'resources/ts/main.tsx'),
                output: {
                    manualChunks(id) {
                        if (/node_modules\/(react|react-dom|react-router|react-router-dom|scheduler)\//.test(id)) return 'react'
                        // Name the chart engine chunk (dashboard and reports only). Not echarts-for-react: it is CJS and its interop helper would make react and echarts import each other.
                        if (/node_modules\/(echarts|zrender)\//.test(id)) return 'echarts'
                    },
                },
            },
        },
        test: {
            // Not CLAUDE.md or AGENTS.md: the coverage report tries to parse every file it includes.
            coverage: {
                include: ['resources/ts/**/*.{ts,tsx}'],
                exclude: ['**/*.stories.*', '**/*.test.*', '**/*.d.ts'],
            },
            projects: [
                {
                    extends: true,
                    test: {
                        name: 'unit',
                        environment: 'node',
                        include: ['resources/ts/**/*.test.{ts,tsx}'],
                    },
                },
                storiesProject('stories-desktop', { width: 1280, height: 800 }, [MOBILE_ONLY]),
                storiesProject('stories-mobile', { width: 390, height: 844 }, [DESKTOP_ONLY]),
                // The matrix. Another language runs at phone size only: the narrowest screen and the longest text.
                ...EXTRA_LOCALES.map((locale) => storiesProject(`stories-mobile-${locale}`, { width: 390, height: 844 }, [DESKTOP_ONLY], { locale })),
                storiesProject('stories-desktop-dark', { width: 1280, height: 800 }, [MOBILE_ONLY], { theme: 'dark' }),
            ],
        },
        server: {
            port: 5173,
            proxy: {
                '/api': 'http://127.0.0.1:8080',
                '/livez': 'http://127.0.0.1:8080',
                '/readyz': 'http://127.0.0.1:8080',
            },
        },
    }
})
