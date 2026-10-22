/// <reference types="vitest/config" />
import path from 'path'
import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { storybookTest } from '@storybook/addon-vitest/vitest-plugin'
import { playwright } from '@vitest/browser-playwright'
import pkg from './package.json'

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
            projects: [
                {
                    extends: true,
                    test: {
                        name: 'unit',
                        environment: 'node',
                        include: ['resources/ts/**/*.test.{ts,tsx}'],
                    },
                },
                // Every story is a test, in a real browser at a desktop and a phone size (`npm run test:stories`).
                {
                    extends: true,
                    plugins: [storybookTest({ configDir: path.join(__dirname, '.storybook') })],
                    // Name every dependency up front; one found mid-run makes Vite reload and fail the run.
                    optimizeDeps: {
                        include: [
                            ...Object.keys(pkg.dependencies).filter((name) => !name.startsWith('@vitejs/')),
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
                        name: 'stories',
                        // Many stories load at once on a cold cache; the Russian ones wait for their language files.
                        testTimeout: 30_000,
                        hookTimeout: 30_000,
                        browser: {
                            enabled: true,
                            headless: true,
                            provider: playwright(),
                            instances: [
                                { browser: 'chromium', name: 'desktop', viewport: { width: 1280, height: 800 } },
                                { browser: 'chromium', name: 'mobile', viewport: { width: 390, height: 844 } },
                            ],
                        },
                    },
                },
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
