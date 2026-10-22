import { useEffect } from 'react'
import type { Decorator, Loader, Preview } from '@storybook/react-vite'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { http, HttpResponse } from 'msw'
import { mswLoader } from 'msw-storybook-addon/csf3'
import { Toaster } from 'sonner'

import i18n, { i18nReady, LOCALE_LABELS, SUPPORTED_LOCALES, type AppLocale } from '@/lib/i18n'
import { baseA11y } from '@/test/a11y'
import '@/index.css'
import './preview.css'

const localeLoader: Loader = async ({ globals }) => {
    await i18nReady
    await i18n.changeLanguage((globals.locale as AppLocale) ?? 'en')
}

const ThemeClass = ({ theme }: { theme: string }) => {
    useEffect(() => {
        const root = document.documentElement
        root.classList.toggle('dark', theme === 'dark')
        root.classList.toggle('light', theme !== 'dark')
        root.style.colorScheme = theme === 'dark' ? 'dark' : 'light'
    }, [theme])
    return null
}

// A fresh query cache per story, so one story's data never leaks into the next.
const withApp: Decorator = (Story, { globals, parameters }) => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } })

    return (
        <QueryClientProvider client={client}>
            <ThemeClass theme={globals.theme} />
            <MemoryRouter initialEntries={[parameters.route ?? '/']}>
                <div className="bg-background text-foreground min-h-8 p-4">
                    <Story />
                </div>
            </MemoryRouter>
            <Toaster position="top-right" richColors />
        </QueryClientProvider>
    )
}

const preview: Preview = {
    decorators: [withApp],
    loaders: [localeLoader, mswLoader()],
    // Any API call a story did not mock fails loudly instead of reaching a real server.
    beforeEach({ msw }) {
        msw.use(http.all('/api/*', ({ request }) => HttpResponse.json({ message: `Unmocked ${request.method} ${new URL(request.url).pathname}` }, { status: 599 })))
    },
    initialGlobals: { locale: 'en', theme: 'light' },
    globalTypes: {
        locale: {
            description: 'Interface language',
            toolbar: {
                icon: 'globe',
                dynamicTitle: true,
                items: SUPPORTED_LOCALES.map((value) => ({ value, title: LOCALE_LABELS[value] })),
            },
        },
        theme: {
            description: 'Color theme',
            toolbar: {
                icon: 'circlehollow',
                dynamicTitle: true,
                items: [
                    { value: 'light', title: 'Light' },
                    { value: 'dark', title: 'Dark' },
                ],
            },
        },
    },
    parameters: {
        layout: 'fullscreen',
        controls: { matchers: { color: /(background|color)$/i, date: /Date$/i } },
        a11y: baseA11y,
        viewport: {
            options: {
                mobile: { name: 'Mobile', styles: { width: '390px', height: '844px' }, type: 'mobile' },
                tablet: { name: 'Tablet', styles: { width: '768px', height: '1024px' }, type: 'tablet' },
            },
        },
    },
}

export default preview
