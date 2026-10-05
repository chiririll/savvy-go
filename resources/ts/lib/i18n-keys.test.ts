import { readdirSync, readFileSync, statSync } from 'node:fs'
import path from 'node:path'
import { describe, expect, it } from 'vitest'

const TS_ROOT = path.resolve(__dirname, '..')
const LOCALES_ROOT = path.join(TS_ROOT, 'locales')
const LOCALES = readdirSync(LOCALES_ROOT)

// i18next plural suffixes; en has two forms, ru four.
const PLURAL_SUFFIX = /_(zero|one|two|few|many|other)$/

/** Leaf keys of a locale file with their text. */
function flatten(value: unknown, prefix = '', out = new Map<string, string>()): Map<string, string> {
    if (value !== null && typeof value === 'object') {
        for (const [key, child] of Object.entries(value)) {
            flatten(child, prefix ? `${prefix}.${key}` : key, out)
        }
    } else {
        out.set(prefix, String(value))
    }
    return out
}

/** Namespace → its leaf keys and texts. */
function loadLocale(locale: string): Map<string, Map<string, string>> {
    const namespaces = new Map<string, Map<string, string>>()
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

/** `{{name}}` and `{{name, format}}` placeholders of a text. */
function placeholders(text: string): string[] {
    return [...text.matchAll(/\{\{\s*([\w.]+)[^}]*\}\}/g)].map((m) => m[1])
}

/**
 * Blanks out comments, keeping every character's position and line, so a
 * `t('…')` mentioned in prose is not taken for a call. Strings are skipped so
 * that `'//'` or a URL inside one is left alone.
 */
function blankComments(source: string): string {
    let out = ''
    let i = 0
    while (i < source.length) {
        const ch = source[i]
        if (ch === '"' || ch === "'" || ch === '`') {
            let j = i + 1
            while (j < source.length && source[j] !== ch) {
                j += source[j] === '\\' ? 2 : 1
            }
            out += source.slice(i, j + 1)
            i = j + 1
        } else if (ch === '/' && (source[i + 1] === '/' || source[i + 1] === '*')) {
            const line = source[i + 1] === '/'
            const close = line ? source.indexOf('\n', i) : source.indexOf('*/', i + 2)
            const stop = close === -1 ? source.length : line ? close : close + 2
            out += source.slice(i, stop).replace(/[^\n]/g, ' ')
            i = stop
        } else {
            out += ch
            i++
        }
    }
    return out
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

const locales = new Map(LOCALES.map((locale) => [locale, loadLocale(locale)]))
const en = locales.get('en')!
const enKeysByNs = new Map([...en].map(([ns, keys]) => [ns, baseKeys(keys.keys())]))
const enKeys = new Set([...enKeysByNs.values()].flatMap((keys) => [...keys]))

interface Usage {
    file: string
    key: string
    /** Set when the call names its namespace: `t('ns:key')` or a `tNs` alias. */
    ns?: string
}

const staticUsages: Usage[] = []
const dynamicUsages: Usage[] = []

for (const file of sourceFiles(TS_ROOT)) {
    const source = readFileSync(file, 'utf8')
    const rel = path.relative(TS_ROOT, file)
    const sourceLines = source.split('\n')

    // `t(…)`, `i18n.t(…)` and aliases such as `const { t: tCommon } = useTranslation('common')`.
    for (const m of blankComments(source).matchAll(/\b(t|t[A-Z]\w*)\(\s*([^)\s][^)]*)/g)) {
        // A deliberately dynamic call says why on its own line or the one above.
        const line = source.slice(0, m.index).split('\n').length - 1
        if (sourceLines.slice(Math.max(0, line - 1), line + 1).some((l) => l.includes('i18n-dynamic'))) {
            continue
        }
        const aliasNs = m[1] === 't' ? undefined : m[1].slice(1).toLowerCase()
        const arg = m[2]
        const literal = /^'([^']*)'|^`([^`$]*)`/.exec(arg)
        if (!literal) {
            dynamicUsages.push({ file: rel, key: `${m[1]}(${arg.trim()}` })
            continue
        }
        const [, explicitNs, key] = /^(?:([a-z]+):)?(.*)$/s.exec(literal[1] ?? literal[2])!
        staticUsages.push({ file: rel, key, ns: explicitNs ?? aliasNs })
    }
}

describe('locale files', () => {
    for (const locale of LOCALES.filter((l) => l !== 'en')) {
        const other = locales.get(locale)!

        it(`${locale} has the same files and keys as en`, () => {
            expect([...other.keys()].sort()).toEqual([...en.keys()].sort())

            for (const [ns, keys] of en) {
                expect(
                    [...baseKeys(other.get(ns)?.keys() ?? [])].sort(),
                    `${locale}/${ns}.json`,
                ).toEqual([...baseKeys(keys.keys())].sort())
            }
        })

        it(`${locale} uses the same placeholders as en`, () => {
            const mismatched: string[] = []
            for (const [ns, enTexts] of en) {
                const byBase = (texts: Map<string, string>) => {
                    const out = new Map<string, Set<string>>()
                    for (const [key, text] of texts) {
                        const base = key.replace(PLURAL_SUFFIX, '')
                        out.set(base, new Set([...(out.get(base) ?? []), ...placeholders(text)]))
                    }
                    return out
                }
                const ours = byBase(enTexts)
                const theirs = byBase(other.get(ns) ?? new Map())
                for (const [key, vars] of ours) {
                    const otherVars = theirs.get(key)
                    if (otherVars && [...vars].sort().join() !== [...otherVars].sort().join()) {
                        mismatched.push(`${ns}:${key}  en {${[...vars]}} vs ${locale} {${[...otherVars]}}`)
                    }
                }
            }
            expect(mismatched).toEqual([])
        })
    }

    for (const locale of LOCALES) {
        it(`${locale} has every plural form the language needs`, () => {
            const required = new Intl.PluralRules(locale).resolvedOptions().pluralCategories
            const incomplete: string[] = []
            for (const [ns, texts] of locales.get(locale)!) {
                const forms = new Map<string, Set<string>>()
                for (const key of texts.keys()) {
                    const m = PLURAL_SUFFIX.exec(key)
                    if (m) {
                        const base = key.slice(0, m.index)
                        forms.set(base, new Set([...(forms.get(base) ?? []), m[1]]))
                    }
                }
                for (const [base, present] of forms) {
                    const missing = required.filter((form) => !present.has(form))
                    if (missing.length > 0) incomplete.push(`${ns}:${base} lacks ${missing.join(', ')}`)
                }
            }
            expect(incomplete).toEqual([])
        })
    }
})

describe('blankComments', () => {
    it('hides calls in comments but keeps code, strings and positions', () => {
        const source = [
            "// t('a.b') in a line comment",
            "const url = 'https://x.y/z' // t('c.d')",
            "/* t('e.f')",
            "   still a comment */ t('g.h')",
        ].join('\n')
        const blanked = blankComments(source)

        expect(blanked).toHaveLength(source.length)
        expect(blanked.split('\n')).toHaveLength(4)
        expect([...blanked.matchAll(/t\('([^']+)'\)/g)].map((m) => m[1])).toEqual(['g.h'])
        expect(blanked).toContain("'https://x.y/z'")
    })
})

describe('translation keys used in code', () => {
    it('finds the call sites it is meant to check', () => {
        expect(staticUsages.length).toBeGreaterThan(100)
        expect(staticUsages.filter((u) => u.ns).length).toBeGreaterThan(50)
    })

    it('static keys exist in en, in the namespace they name', () => {
        const missing = staticUsages
            .filter(({ key, ns }) => !(ns && enKeysByNs.has(ns) ? enKeysByNs.get(ns)!.has(key) : enKeys.has(key)))
            .map(({ file, key, ns }) => `${ns ? `${ns}:` : ''}${key}  (${file})`)
        expect(missing).toEqual([])
    })

    // A key built at runtime cannot be checked against the locale files, so a
    // renamed or dropped translation goes unnoticed. Spell every key out, or
    // mark an unavoidable call with an `i18n-dynamic: reason` comment.
    it('has no dynamic keys', () => {
        const dynamic = dynamicUsages.map(({ file, key }) => `${key}…)  (${file})`)
        expect(dynamic).toEqual([])
    })
})
