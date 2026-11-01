/**
 * Search keywords for the transaction list: `account:"Cash USD" tags:food,coffee`.
 * They are parsed here into the structured API params; the API stays id-based
 * and `q` stays plain text.
 */

export type SearchType = 'income' | 'expense' | 'transfer'

export interface SearchRef {
    id: number
    /** Every name the entity answers to; the first one is shown and written back. */
    names: string[]
}

export interface SearchRefs {
    accounts: SearchRef[]
    categories: SearchRef[]
    tags: SearchRef[]
    currencies: string[]
}

export interface ParsedSearch {
    q: string
    accountIds: number[]
    currency: string[]
    categoryIds: number[]
    tagIds: number[]
    type: SearchType | null
}

export interface SearchCompletion {
    label: string
    /** The whole input after accepting this suggestion. */
    insert: string
}

const TYPES: SearchType[] = ['income', 'expense', 'transfer']
const KEYWORDS = ['account', 'currency', 'category', 'tags', 'type'] as const
type Keyword = (typeof KEYWORDS)[number]

interface Token {
    raw: string
    start: number
}

/** Splits on whitespace outside double quotes; an unterminated quote runs to the end. */
function tokenize(input: string): Token[] {
    const tokens: Token[] = []
    let start = -1
    let quoted = false
    for (let i = 0; i <= input.length; i++) {
        const ch = input[i]
        const boundary = ch === undefined || (!quoted && /\s/.test(ch))
        if (boundary) {
            if (start >= 0) {
                tokens.push({ raw: input.slice(start, i), start })
                start = -1
            }
            continue
        }
        if (start < 0) start = i
        if (ch === '"') quoted = !quoted
    }
    return tokens
}

/** Splits a keyword value on commas outside quotes and strips the quotes. */
function splitItems(value: string): string[] {
    const items: string[] = []
    let current = ''
    let quoted = false
    for (const ch of value) {
        if (ch === '"') {
            quoted = !quoted
        } else if (ch === ',' && !quoted) {
            items.push(current)
            current = ''
        } else {
            current += ch
        }
    }
    items.push(current)
    return items.map((item) => item.trim()).filter(Boolean)
}

function splitKeyword(raw: string): { key: Keyword; value: string } | null {
    const colon = raw.indexOf(':')
    if (colon <= 0) return null
    const key = raw.slice(0, colon).toLowerCase()
    return (KEYWORDS as readonly string[]).includes(key) ? { key: key as Keyword, value: raw.slice(colon + 1) } : null
}

function findRefs(list: SearchRef[], items: string[]): number[] | null {
    const ids: number[] = []
    for (const item of items) {
        const wanted = item.toLowerCase()
        const ref = list.find((entry) => entry.names.some((name) => name.toLowerCase() === wanted))
        if (!ref) return null
        if (!ids.includes(ref.id)) ids.push(ref.id)
    }
    return ids
}

function addAll<T>(target: T[], values: T[]) {
    for (const value of values) {
        if (!target.includes(value)) target.push(value)
    }
}

export function parseTransactionSearch(input: string, refs: SearchRefs): ParsedSearch {
    const parsed: ParsedSearch = { q: '', accountIds: [], currency: [], categoryIds: [], tagIds: [], type: null }
    const text: string[] = []

    for (const { raw } of tokenize(input)) {
        const keyword = splitKeyword(raw)
        const items = keyword ? splitItems(keyword.value) : []
        let resolved = false

        if (keyword && items.length > 0) {
            switch (keyword.key) {
                case 'account': {
                    const ids = findRefs(refs.accounts, items)
                    if (ids) addAll(parsed.accountIds, ids)
                    resolved = ids !== null
                    break
                }
                case 'category': {
                    const ids = findRefs(refs.categories, items)
                    if (ids) addAll(parsed.categoryIds, ids)
                    resolved = ids !== null
                    break
                }
                case 'tags': {
                    const ids = findRefs(refs.tags, items)
                    if (ids) addAll(parsed.tagIds, ids)
                    resolved = ids !== null
                    break
                }
                case 'currency': {
                    const codes = items.map((item) => refs.currencies.find((code) => code.toLowerCase() === item.toLowerCase()))
                    resolved = codes.every(Boolean)
                    if (resolved) addAll(parsed.currency, codes as string[])
                    break
                }
                case 'type': {
                    const last = items[items.length - 1].toLowerCase()
                    const type = TYPES.find((candidate) => candidate === last)
                    resolved = items.every((item) => TYPES.includes(item.toLowerCase() as SearchType))
                    if (resolved && type) parsed.type = type
                    break
                }
            }
        }

        if (resolved) continue
        text.push(keyword ? raw : raw.replace(/"/g, ''))
    }

    parsed.q = text.filter(Boolean).join(' ')
    return parsed
}

function quoteIfNeeded(value: string): string {
    return /[\s,"]/.test(value) ? `"${value.replace(/"/g, '')}"` : value
}

function nameOf(list: SearchRef[], id: number): string | null {
    return list.find((entry) => entry.id === id)?.names[0] ?? null
}

function keywordText(key: Keyword, values: Array<string | null>): string {
    const names = values.filter((value): value is string => value !== null).map(quoteIfNeeded)
    return names.length > 0 ? `${key}:${names.join(',')}` : ''
}

/** Writes filters back as the keywords that parse to them (ids that no longer exist are dropped). */
export function serializeTransactionSearch(parsed: ParsedSearch, refs: SearchRefs): string {
    return [
        parsed.q,
        parsed.type ? `type:${parsed.type}` : '',
        keywordText('account', parsed.accountIds.map((id) => nameOf(refs.accounts, id))),
        keywordText('currency', parsed.currency),
        keywordText('category', parsed.categoryIds.map((id) => nameOf(refs.categories, id))),
        keywordText('tags', parsed.tagIds.map((id) => nameOf(refs.tags, id))),
    ]
        .filter(Boolean)
        .join(' ')
}

function valueCandidates(key: Keyword, refs: SearchRefs): string[] {
    switch (key) {
        case 'account':
            return refs.accounts.flatMap((ref) => ref.names)
        case 'category':
            return refs.categories.flatMap((ref) => ref.names)
        case 'tags':
            return refs.tags.flatMap((ref) => ref.names)
        case 'currency':
            return refs.currencies
        case 'type':
            return TYPES
    }
}

/** Suggestions for the word being typed at the end of the input: keywords, then their values. */
export function suggestSearchCompletions(input: string, refs: SearchRefs): SearchCompletion[] {
    if (input === '' || /\s$/.test(input)) return []
    const tokens = tokenize(input)
    const last = tokens[tokens.length - 1]
    if (!last) return []
    const before = input.slice(0, last.start)
    const colon = last.raw.indexOf(':')

    if (colon < 0) {
        const typed = last.raw.toLowerCase()
        if (typed.includes('"')) return []
        return KEYWORDS.filter((key) => key.startsWith(typed) && key !== typed).map((key) => ({
            label: `${key}:`,
            insert: `${before}${key}:`,
        }))
    }

    const keyword = splitKeyword(last.raw)
    if (!keyword) return []
    const value = keyword.value
    // The item being typed starts after the last comma outside quotes.
    let cut = 0
    let quoted = false
    for (let i = 0; i < value.length; i++) {
        if (value[i] === '"') quoted = !quoted
        else if (value[i] === ',' && !quoted) cut = i + 1
    }
    const done = value.slice(0, cut)
    const typed = value.slice(cut).replace(/"/g, '').toLowerCase()
    const used = splitItems(done).map((item) => item.toLowerCase())
    const seen = new Set<string>()

    return valueCandidates(keyword.key, refs)
        .filter((name) => {
            const lower = name.toLowerCase()
            if (seen.has(lower) || used.includes(lower) || !lower.startsWith(typed) || lower === typed) return false
            seen.add(lower)
            return true
        })
        .map((name) => ({
            label: name,
            insert: `${before}${last.raw.slice(0, colon + 1)}${done}${quoteIfNeeded(name)} `,
        }))
}
