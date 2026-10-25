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

/** A short path to an element, e.g. `div#root > main.flex > a.truncate`, so it can be found in the DOM. */
function describe(el: Element): string {
    const part = (node: Element) => {
        const classes = String(node.getAttribute('class') ?? '').split(/\s+/).filter(Boolean).slice(0, 3)
        return node.tagName.toLowerCase() + (node.id ? `#${node.id}` : '') + classes.map((c) => `.${c}`).join('')
    }
    const path: string[] = []
    for (let node: Element | null = el; node && node !== document.body && path.length < 4; node = node.parentElement) {
        path.unshift(part(node))
    }
    const text = (el.textContent ?? '').trim().replace(/\s+/g, ' ').slice(0, 40)
    return `${path.join(' > ')}${text ? ` "${text}"` : ''}`
}

/** The elements that make the page wider than the window, widest first, as a path, text and right edge. */
function widest(windowWidth: number): string {
    const root = document.documentElement
    // A box that scrolls or clips by itself keeps what sticks out of it from widening the page. The body and the root are
    // skipped: their overflow is handed to the window, which is the very scroll bar being looked for.
    const clipped = (el: Element) => {
        for (let parent = el.parentElement; parent && parent !== document.body && parent !== root; parent = parent.parentElement) {
            if (getComputedStyle(parent).overflowX !== 'visible') return true
        }
        return false
    }
    const right = (el: Element) => el.getBoundingClientRect().right
    const sticksOut = (el: Element) => right(el) > windowWidth + 0.5
    const all = [...document.body.querySelectorAll('*')].filter(sticksOut)
    // Only the innermost ones: a parent is wide because of the child that makes it so.
    const innermost = all.filter((el) => ![...el.children].some(sticksOut))
    const visible = innermost.filter((el) => !clipped(el))
    // Everything may be clipped yet the page still scroll (a fixed or transformed box, a pseudo element): list them all then.
    const found = (visible.length > 0 ? visible : innermost).sort((a, b) => right(b) - right(a)).slice(0, 3)
    if (found.length === 0) {
        return `unknown (document scrollWidth ${root.scrollWidth}, body scrollWidth ${document.body.scrollWidth})`
    }
    return found.map((el) => `${describe(el)} (right edge ${Math.round(right(el))}px)`).join('; ')
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
