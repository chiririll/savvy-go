import { readdirSync, readFileSync, statSync } from 'node:fs'
import path from 'node:path'
import { describe, expect, it } from 'vitest'

const TS_ROOT = path.resolve(__dirname, '..')
const LOCALES_ROOT = path.join(TS_ROOT, 'locales')
const LOCALES = readdirSync(LOCALES_ROOT)

// i18next plural suffixes; en has two forms, ru four.
const PLURAL_SUFFIX = /_(zero|one|two|few|many|other)$/

function flatten(value: unknown, prefix = '', out = new Set<string>()): Set<string> {
    if (value !== null && typeof value === 'object') {
        for (const [key, child] of Object.entries(value)) {
            flatten(child, prefix ? `${prefix}.${key}` : key, out)
        }
    } else {
        out.add(prefix)
    }
    return out
}

function loadLocale(locale: string): Map<string, Set<string>> {
    const namespaces = new Map<string, Set<string>>()
    for (const file of readdirSync(path.join(LOCALES_ROOT, locale))) {
        const json = JSON.parse(readFileSync(path.join(LOCALES_ROOT, locale, file), 'utf8'))
        namespaces.set(file.replace(/\.json$/, ''), flatten(json))
    }
    return namespaces
}

/** Plural variants collapse into their base key. */
function baseKeys(keys: Iterable<string>): Set<string> {
    return new Set([...keys].map((k) => k.replace(PLURAL_SUFFIX, '')))
}

function sourceFiles(dir: string): string[] {
    return readdirSync(dir).flatMap((name) => {
        const full = path.join(dir, name)
        if (statSync(full).isDirectory()) {
            return name === 'locales' ? [] : sourceFiles(full)
        }
        return /\.tsx?$/.test(name) && !/\.test\.tsx?$/.test(name) ? [full] : []
    })
}

const en = loadLocale('en')
const enKeys = baseKeys([...en.values()].flatMap((keys) => [...keys]))

/** Keys in a `t('ns:a.b')` call; the namespace prefix is dropped. */
const stripNamespace = (key: string) => key.replace(/^[a-z]+:/, '')

interface Usage {
    file: string
    key: string
}

const staticUsages: Usage[] = []
const dynamicUsages: Usage[] = []

for (const file of sourceFiles(TS_ROOT)) {
    const source = readFileSync(file, 'utf8')
    const rel = path.relative(TS_ROOT, file)
    const sourceLines = source.split('\n')

    for (const m of source.matchAll(/\bt\(\s*([^)\s][^)]*)/g)) {
        // A deliberately dynamic call says why on its own line or the one above.
        const line = source.slice(0, m.index).split('\n').length - 1
        if (sourceLines.slice(Math.max(0, line - 1), line + 1).some((l) => l.includes('i18n-dynamic'))) {
            continue
        }
        const arg = m[1]
        const literal = /^'([^']*)'|^`([^`$]*)`/.exec(arg)
        if (literal) {
            staticUsages.push({ file: rel, key: stripNamespace(literal[1] ?? literal[2]) })
        } else {
            dynamicUsages.push({ file: rel, key: arg.trim() })
        }
    }
}

describe('locale parity', () => {
    for (const locale of LOCALES.filter((l) => l !== 'en')) {
        it(`${locale} has the same files and keys as en`, () => {
            const other = loadLocale(locale)
            expect([...other.keys()].sort()).toEqual([...en.keys()].sort())

            for (const [ns, keys] of en) {
                expect(
                    [...baseKeys(other.get(ns) ?? [])].sort(),
                    `${locale}/${ns}.json`,
                ).toEqual([...baseKeys(keys)].sort())
            }
        })
    }
})

describe('translation keys used in code', () => {
    it('finds the call sites it is meant to check', () => {
        expect(staticUsages.length).toBeGreaterThan(100)
    })

    it('static keys exist in en', () => {
        const missing = staticUsages
            .filter(({ key }) => !enKeys.has(key))
            .map(({ file, key }) => `${key}  (${file})`)
        expect(missing).toEqual([])
    })

    // A key built at runtime cannot be checked against the locale files, so a
    // renamed or dropped translation goes unnoticed. Spell every key out, or
    // mark an unavoidable call with an `i18n-dynamic: reason` comment.
    it('has no dynamic keys', () => {
        const dynamic = dynamicUsages.map(({ file, key }) => `t(${key}…)  (${file})`)
        expect(dynamic).toEqual([])
    })
})
