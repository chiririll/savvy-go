// How a browser context is set up for a shot: the same for the page capture (run.ts) and the
// scenarios (scenarios/), so both show the app alike.
import { devices, type Browser, type BrowserContextOptions } from 'playwright'

export const timezoneId = process.env.TZ?.trim() || 'UTC'

export const viewports: Record<string, BrowserContextOptions> = {
    desktop: { viewport: { width: 1440, height: 900 } },
    mobile: { ...devices['Pixel 7'] },
}

// Animations are switched off rather than waited for.
const noMotion = `*, *::before, *::after {
    animation: none !important;
    transition: none !important;
    caret-color: transparent !important;
}`

export interface ContextSpec {
    baseURL: string
    viewport: string
    language: string
    theme: string
    storageState?: BrowserContextOptions['storageState']
    /** The space the app opens (its choice is kept in the browser); none: the app picks. */
    spaceId?: number
    /** Moves the browser's clock to this day; time keeps running (charts stall on a frozen one). */
    now?: Date | null
}

export async function newContext(browser: Browser, spec: ContextSpec) {
    const context = await browser.newContext({
        ...viewports[spec.viewport], baseURL: spec.baseURL, timezoneId, storageState: spec.storageState,
        // What the browser itself says; the app also keeps its own choice in storage (below).
        locale: spec.language, colorScheme: spec.theme as 'light' | 'dark',
        reducedMotion: 'reduce',
    })
    if (spec.now) await context.clock.install({ time: spec.now })
    await context.addInitScript(
        ({ css, spaceId, language, theme }) => {
            if (spaceId !== undefined) {
                localStorage.setItem('savvy-space', JSON.stringify({ state: { currentId: spaceId }, version: 0 }))
            }
            // The keys of the app's language and theme choice (lib/i18n.ts, hooks/use-theme.ts).
            localStorage.setItem('savvy.locale', language)
            localStorage.setItem('theme', theme)
            document.addEventListener('DOMContentLoaded', () => {
                const style = document.createElement('style')
                style.textContent = css
                document.head.append(style)
            })
        },
        { css: noMotion, spaceId: spec.spaceId, language: spec.language, theme: spec.theme },
    )
    return context
}
