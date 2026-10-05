import { useSyncExternalStore } from 'react'
import { flushSync } from 'react-dom'

export type Theme = 'light' | 'dark'
export type ThemePreference = Theme | 'auto'

type ThemeState = {
    preference: ThemePreference
    theme: Theme
}

const STORAGE_KEY = 'theme'

const SERVER_STATE: ThemeState = { preference: 'auto', theme: 'light' }

function getSystemTheme(): Theme {
    if (typeof window === 'undefined') return 'light'
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function parsePreference(value: string | null): ThemePreference {
    if (value === 'light' || value === 'dark' || value === 'auto') return value
    return 'auto'
}

function resolveTheme(preference: ThemePreference): Theme {
    return preference === 'auto' ? getSystemTheme() : preference
}

function readDomTheme(): Theme {
    if (typeof document === 'undefined') return 'light'
    // The blocking inline script in the document head resolves and applies the
    // theme class before first paint; the DOM is the single source of truth.
    return document.documentElement.classList.contains('dark') ? 'dark' : 'light'
}

function readInitialState(): ThemeState {
    if (typeof window === 'undefined') return SERVER_STATE
    const preference = parsePreference(localStorage.getItem(STORAGE_KEY))
    return { preference, theme: readDomTheme() }
}

let state: ThemeState = readInitialState()
const listeners = new Set<() => void>()

function applyTheme(theme: Theme) {
    const root = document.documentElement
    root.classList.toggle('dark', theme === 'dark')
    root.classList.toggle('light', theme === 'light')
    root.style.colorScheme = theme
    document
        .querySelector('meta[name="theme-color"]')
        ?.setAttribute('content', theme === 'dark' ? '#1a1a1a' : '#ffffff')
}

function notifyAll() {
    listeners.forEach((notify) => notify())
}

export type ThemeOrigin = { x: number; y: number }

const WAVE_DURATION_MS = 800
const WAVE_POINTS = 120
const WAVE_LOBES = 7
const WAVE_AMPLITUDE_PX = 45
const CLIP_VAR = '--theme-clip'
const DURATION_VAR = '--theme-duration'

let activeTransition: ViewTransition | null = null
let activeFrame = 0

const easeInOutCubic = (t: number) => (t < 0.5 ? 4 * t * t * t : 1 - (-2 * t + 2) ** 3 / 2)

/** Circle whose edge ripples; the ripple fades out at the start and end so it lands on a clean full cover. */
function waveClip(origin: ThemeOrigin, maxRadius: number, t: number): string {
    const radius = easeInOutCubic(t) * maxRadius
    const amplitude = WAVE_AMPLITUDE_PX * Math.sin(Math.PI * t)
    const phase = t * Math.PI * 4
    const points: string[] = []
    for (let i = 0; i < WAVE_POINTS; i++) {
        const angle = (i / WAVE_POINTS) * Math.PI * 2
        const r = Math.max(0, radius + amplitude * Math.sin(angle * WAVE_LOBES + phase))
        points.push(`${origin.x + r * Math.cos(angle)}px ${origin.y + r * Math.sin(angle)}px`)
    }
    return `polygon(${points.join(',')})`
}

function runWave(transition: ViewTransition, origin: ThemeOrigin) {
    const root = document.documentElement
    const maxRadius =
        Math.hypot(Math.max(origin.x, window.innerWidth - origin.x), Math.max(origin.y, window.innerHeight - origin.y)) +
        WAVE_AMPLITUDE_PX
    const startedAt = performance.now()

    const tick = (now: number) => {
        const t = Math.min(1, (now - startedAt) / WAVE_DURATION_MS)
        root.style.setProperty(CLIP_VAR, waveClip(origin, maxRadius, t))
        if (t < 1) activeFrame = requestAnimationFrame(tick)
    }

    transition.ready.then(() => {
        activeFrame = requestAnimationFrame(tick)
    })
}

function canAnimate(): boolean {
    return (
        typeof document.startViewTransition === 'function' &&
        !window.matchMedia('(prefers-reduced-motion: reduce)').matches
    )
}

function commit(next: ThemeState, origin?: ThemeOrigin) {
    if (next.preference === state.preference && next.theme === state.theme) return
    const themeChanged = next.theme !== state.theme
    state = next

    const apply = () => {
        applyTheme(next.theme)
        notifyAll()
    }

    if (!themeChanged || !canAnimate()) {
        apply()
        return
    }

    activeTransition?.skipTransition()
    const root = document.documentElement
    root.dataset.themeTransition = origin ? 'wave' : 'fade'
    if (origin) {
        root.style.setProperty(CLIP_VAR, `circle(0px at ${origin.x}px ${origin.y}px)`)
        // Slightly longer than the wave so a late frame never outlives the transition.
        root.style.setProperty(DURATION_VAR, `${WAVE_DURATION_MS + 100}ms`)
    }
    // Charts read the theme from React state, so flush it before the "after" snapshot is taken.
    const transition = document.startViewTransition(() => flushSync(apply))
    activeTransition = transition
    if (origin) runWave(transition, origin)

    transition.finished.finally(() => {
        // A skipped transition settles after its replacement has started; leave the newer one's state alone.
        if (activeTransition !== transition) return
        activeTransition = null
        cancelAnimationFrame(activeFrame)
        delete root.dataset.themeTransition
        root.style.removeProperty(CLIP_VAR)
        root.style.removeProperty(DURATION_VAR)
    })
}

function setPreference(preference: ThemePreference, origin?: ThemeOrigin) {
    localStorage.setItem(STORAGE_KEY, preference)
    commit({ preference, theme: resolveTheme(preference) }, origin)
}

function syncFromSystem() {
    if (state.preference !== 'auto') return
    commit({ preference: 'auto', theme: getSystemTheme() })
}

if (typeof window !== 'undefined') {
    window.addEventListener('storage', (event) => {
        if (event.key !== STORAGE_KEY) return
        const preference = parsePreference(event.newValue)
        commit({ preference, theme: resolveTheme(preference) })
    })

    const media = window.matchMedia('(prefers-color-scheme: dark)')
    media.addEventListener('change', syncFromSystem)
}

function subscribe(notify: () => void): () => void {
    listeners.add(notify)
    return () => {
        listeners.delete(notify)
    }
}

function getSnapshot(): ThemeState {
    return state
}

export function useTheme() {
    const snapshot = useSyncExternalStore(subscribe, getSnapshot, () => SERVER_STATE)

    return {
        theme: snapshot.theme,
        preference: snapshot.preference,
        setTheme: setPreference,
    }
}
