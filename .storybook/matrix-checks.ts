// Checks for the language and theme matrix (`npm run test:stories:full`, see docs/development.md). They run after every
// story whose `locale` is not English or whose `theme` is not light, so a story never has to ask for them.
import i18n from '@/lib/i18n'

const PROCESSOR = 'story-missing-translation'

/** A namespaced key shown as text, e.g. `forms:save`: `t()` found nothing and returned the key. */
const RAW_KEY = /\b(?:common|nav|auth|settings|pages|forms|defaults):[A-Za-z_][\w.]*/g

const missing = new Set<string>()

// Runs on every t() result. A key the active language lacks is answered from English, which looks fine on screen, so
// ask the active language alone.
i18n.use({
    type: 'postProcessor',
    name: PROCESSOR,
    process(value: string, key: string | string[], options: { lng?: string }, translator: { exists: (k: string, o: object) => boolean }) {
        const lng = options.lng ?? i18n.resolvedLanguage ?? i18n.language
        const keys = Array.isArray(key) ? key : [key]
        if (!keys.some((k) => translator.exists(k, { ...options, lng, fallbackLng: false }))) {
            missing.add(`${lng}: ${keys[0]}`)
        }
        return value
    },
})

export function isMatrixRun(globals: Record<string, unknown>): boolean {
    if (!import.meta.env.STORYBOOK_MATRIX_CHECKS) return false
    return (globals.locale ?? 'en') !== 'en' || (globals.theme ?? 'light') !== 'light'
}

export function startMatrixChecks(context: { globals: Record<string, unknown> }) {
    missing.clear()
    i18n.options.postProcess = isMatrixRun(context.globals) ? [PROCESSOR] : []
}

/** The elements that stick out of the window and are not inside a scrolling or clipping box, as tag.class (right edge). */
function widest(windowWidth: number): string {
    const clipped = (el: Element) => {
        for (let parent = el.parentElement; parent && parent !== document.documentElement; parent = parent.parentElement) {
            if (getComputedStyle(parent).overflowX !== 'visible') return true
        }
        return false
    }
    const sticksOut = (el: Element) => el.getBoundingClientRect().right > windowWidth
    // Only the innermost ones: a parent is wide because of the child that makes it so.
    const found = [...document.body.querySelectorAll('*')]
        .filter((el) => sticksOut(el) && !clipped(el) && ![...el.children].some(sticksOut))
        .map((el) => ({ el, right: Math.round(el.getBoundingClientRect().right) }))
        .sort((a, b) => b.right - a.right)
        .slice(0, 3)
    return (
        found
            .map(({ el, right }) => `${el.tagName.toLowerCase()}.${String(el.className).slice(0, 50).replace(/ +/g, '.')} (${right}px)`)
            .join(', ') || 'unknown'
    )
}

/** Throws when the story shows a raw key, uses a string the language lacks, or scrolls the page sideways. */
export function finishMatrixChecks(globals: Record<string, unknown>) {
    if (!isMatrixRun(globals)) return

    const problems: string[] = []

    if (missing.size > 0) {
        problems.push(`missing translation (${missing.size}): ${[...missing].slice(0, 5).join(', ')}`)
    }

    const rawKeys = new Set((document.body.innerText.match(RAW_KEY) ?? []).slice(0, 5))
    if (rawKeys.size > 0) {
        problems.push(`raw translation key shown: ${[...rawKeys].join(', ')}`)
    }

    const root = document.documentElement
    if (root.scrollWidth > root.clientWidth) {
        problems.push(`horizontal overflow: page is ${root.scrollWidth}px wide in a ${root.clientWidth}px window, widest: ${widest(root.clientWidth)}`)
    }

    if (problems.length > 0) {
        throw new Error(`Matrix check (${globals.locale}, ${globals.theme}): ${problems.join('; ')}`)
    }
}
