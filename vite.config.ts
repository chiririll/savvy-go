import path from 'path'
import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

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
        },
        build: {
            outDir: 'public/build',
            emptyOutDir: true,
            manifest: true,
            rollupOptions: {
                input: path.resolve(__dirname, 'resources/ts/main.tsx'),
            },
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
