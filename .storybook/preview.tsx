import './pin-locale'
import { useEffect } from 'react'
import type { Decorator, Loader, Preview } from '@storybook/react-vite'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { NuqsTestingAdapter } from 'nuqs/adapters/testing'
import { mswLoader } from 'msw-storybook-addon/csf3'
import { Toaster } from 'sonner'
import { configure } from 'storybook/test'

import i18n, { i18nReady, LOCALE_LABELS, SUPPORTED_LOCALES, type AppLocale } from '@/lib/i18n'
import { baseA11y } from '@/test/a11y'
import { setUpApp } from '@/test/app-state'
import { finishMatrixChecks, startMatrixChecks } from './matrix-checks'
import { AbilityProvider } from '@/components/providers/AbilityProvider'
import { ReadOnlyProvider } from '@/components/providers/ReadOnlyProvider'
import '@/index.css'
import './preview.css'

// With many stories loading at once a mocked request can take over a second, the default wait of a findBy query.
configure({ asyncUtilTimeout: 5000 })

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
                <NuqsTestingAdapter searchParams={(parameters.route ?? '/').split('?')[1] ?? ''} hasMemory>
                    <AbilityProvider>
                        <ReadOnlyProvider>
                            <div className="bg-background text-foreground min-h-8 p-4">
                                <Story />
                            </div>
                        </ReadOnlyProvider>
                    </AbilityProvider>
                </NuqsTestingAdapter>
            </MemoryRouter>
            <Toaster position="top-right" richColors />
        </QueryClientProvider>
    )
}

const preview: Preview = {
    decorators: [withApp],
    loaders: [localeLoader, mswLoader()],
    beforeEach(context) {
        const { msw, parameters } = context
        startMatrixChecks(context)
        return setUpApp(parameters.app, (...handlers) => msw.use(...handlers))
    },
    afterEach({ globals }) {
        finishMatrixChecks(globals)
    },
    // English and light, unless the vitest project asks for another pair (vite.config.ts; the language and theme matrix).
    initialGlobals: {
        locale: import.meta.env.STORYBOOK_LOCALE ?? 'en',
        theme: import.meta.env.STORYBOOK_THEME ?? 'light',
        viewport: import.meta.env.STORYBOOK_VIEWPORT ? { value: import.meta.env.STORYBOOK_VIEWPORT, isRotated: false } : undefined,
    },
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
                desktop: { name: 'Desktop', styles: { width: '1280px', height: '800px' }, type: 'desktop' },
                mobile: { name: 'Mobile', styles: { width: '390px', height: '844px' }, type: 'mobile' },
                tablet: { name: 'Tablet', styles: { width: '768px', height: '1024px' }, type: 'tablet' },
            },
        },
    },
}

export default preview
