// What to screenshot. Without a name that is everything, in English and the light theme; a name
// picks a file of configs/ (configs/<name>.json). See docs/scripts.md. In a file, a list that is
// left out or empty takes everything for users, spaces, viewports and pages, and the default for
// languages and themes.
import { existsSync, readdirSync } from 'node:fs'
import { readFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import type { Manifest } from './manifest.ts'

const here = path.dirname(fileURLToPath(import.meta.url))

/** The languages of the app: a directory of translations each. */
export const languages = readdirSync(path.join(here, '..', '..', 'resources', 'ts', 'locales'), { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .map((entry) => entry.name)
    .sort()

export const themes = ['light', 'dark']

export interface Config {
    /** Keys of the seeded users (the manifest's); empty: all of them. */
    users: string[]
    /** Names of spaces; empty: every space of each user. */
    spaces: string[]
    /** Empty: all viewports. */
    viewports: string[]
    /** Empty: English. */
    languages: string[]
    /** Empty: light. */
    themes: string[]
    /** Paths as in resources/ts/app/pages.ts; "/settings" takes the pages under it. Empty: all pages. */
    pages: string[]
    /** On a phone, the sidebar drawer opened. */
    sidebar: boolean
    /** The dialogs of each page. */
    dialogs: boolean
    /** The tabs of each page and dialog. */
    tabs: boolean
}

export const configsDir = path.join(here, 'configs')

/** The names of the configs there are. */
export const configNames = () =>
    existsSync(configsDir)
        ? readdirSync(configsDir)
              .filter((f) => f.endsWith('.json'))
              .map((f) => f.slice(0, -'.json'.length))
              .sort()
        : []

/** Whether a page, by its path as in pages.ts, is one the config asks for. */
export const pageWanted = (config: Config, template: string) =>
    config.pages.length === 0 ||
    config.pages.some((p) => {
        // "/settings" is that page and the ones under it; "/" is the dashboard alone, not every page.
        const base = p.replace(/\/+$/, '')
        return template === (base || '/') || (base !== '' && template.startsWith(`${base}/`))
    })

/**
 * Reads and checks the config called name (nothing: the defaults); the error says what is wrong
 * with which setting.
 */
export async function loadConfig(name: string, manifest: Manifest, known: { viewports: string[]; pages: string[] }): Promise<Config> {
    let raw: Record<string, unknown> = {}
    if (name) {
        const file = path.join(configsDir, `${name.replace(/\.json$/, '')}.json`)
        if (!existsSync(file)) throw new Error(`no config "${name}"; there are ${configNames().join(', ') || 'none'}`)
        try {
            raw = JSON.parse(await readFile(file, 'utf8'))
        } catch (e) {
            throw new Error(`config ${name}: ${(e as Error).message}`)
        }
    }

    const list = (key: string): string[] => {
        const value = raw[key]
        if (value === undefined) return []
        if (!Array.isArray(value) || value.some((v) => typeof v !== 'string')) throw new Error(`config "${key}": a list of strings is expected`)
        return value as string[]
    }
    const flag = (key: string): boolean => {
        const value = raw[key]
        if (value === undefined) return true
        if (typeof value !== 'boolean') throw new Error(`config "${key}": true or false is expected`)
        return value
    }
    const oneOf = (key: string, wanted: string[], options: string[], whenEmpty: string[]) => {
        const bad = wanted.filter((w) => !options.includes(w))
        if (bad.length) throw new Error(`config "${key}": unknown ${bad.map((b) => `"${b}"`).join(', ')}; there are ${options.join(', ')}`)
        return wanted.length ? wanted : whenEmpty
    }

    const users = list('users')
    const spaces = list('spaces')
    const pages = list('pages')

    const userKeys = manifest.users.map((u) => u.key)
    const badUsers = users.filter((u) => !userKeys.includes(u))
    if (badUsers.length) throw new Error(`config "users": unknown ${badUsers.map((b) => `"${b}"`).join(', ')}; there are ${userKeys.join(', ')}`)

    const spaceNames = manifest.spaces.map((s) => s.name)
    const badSpaces = spaces.filter((s) => !spaceNames.some((n) => n.toLowerCase() === s.toLowerCase()))
    if (badSpaces.length) throw new Error(`config "spaces": unknown ${badSpaces.map((b) => `"${b}"`).join(', ')}; there are ${spaceNames.join(', ')}`)

    const config: Config = {
        users,
        spaces,
        viewports: oneOf('viewports', list('viewports'), known.viewports, known.viewports),
        languages: oneOf('languages', list('languages'), languages, ['en']),
        themes: oneOf('themes', list('themes'), themes, ['light']),
        pages,
        sidebar: flag('sidebar'),
        dialogs: flag('dialogs'),
        tabs: flag('tabs'),
    }

    // A path that matches nothing is most likely a typo.
    const lost = pages.filter((p) => !known.pages.some((t) => pageWanted({ ...config, pages: [p] }, t)))
    if (lost.length) throw new Error(`config "pages": nothing at ${lost.map((p) => `"${p}"`).join(', ')}; the pages are ${known.pages.join(', ')}`)
    return config
}
