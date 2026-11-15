import type { CurrencyCatalogItem } from '@/types'

/** How many currencies the picker lists at once; typing narrows the rest down. */
export const MAX_CURRENCY_OPTIONS = 50

/** The shape the backend accepts for a currency code (spacefill's `currencyCodePattern`). */
const CODE = /^[A-Z0-9]{2,10}$/

interface Options {
    /** Codes to leave out, such as the ones already chosen. */
    exclude?: ReadonlySet<string>
    /** Listed first when nothing is typed. */
    popular?: readonly string[]
}

/** 0: the code starts with the query, 1: the code contains it, 2: the name or symbol does. -1: no match. */
function rank(c: CurrencyCatalogItem, q: string): number {
    const code = c.code.toLowerCase()
    if (code.startsWith(q)) return 0
    if (code.includes(q)) return 1
    return c.name.toLowerCase().includes(q) || c.symbol.toLowerCase().includes(q) ? 2 : -1
}

/** The currencies matching `query` by code, name or symbol, best first and capped; `total` counts all matches. */
export function searchCurrencies(
    catalog: readonly CurrencyCatalogItem[],
    query: string,
    { exclude, popular = [] }: Options = {},
): { shown: CurrencyCatalogItem[]; total: number } {
    const q = query.trim().toLowerCase()
    const free = catalog.filter((c) => !exclude?.has(c.code))
    let matches: CurrencyCatalogItem[]
    if (q) {
        matches = free
            .map((c) => ({ c, r: rank(c, q) }))
            .filter(({ r }) => r >= 0)
            .sort((a, b) => a.r - b.r)
            .map(({ c }) => c)
    } else {
        const first = popular.flatMap((code) => free.filter((c) => c.code === code))
        matches = [...first, ...free.filter((c) => !popular.includes(c.code))]
    }
    return { shown: matches.slice(0, MAX_CURRENCY_OPTIONS), total: matches.length }
}

/** The typed text as a currency code the list does not have, or null: for a short list, where any code may be typed. */
export function manualCode(catalog: readonly CurrencyCatalogItem[], query: string): string | null {
    const code = query.trim().toUpperCase()
    return CODE.test(code) && !catalog.some((c) => c.code === code) ? code : null
}
