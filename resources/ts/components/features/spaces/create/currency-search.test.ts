import { describe, expect, it } from 'vitest'
import type { CurrencyCatalogItem } from '@/types'
import { MAX_CURRENCY_OPTIONS, manualCode, searchCurrencies } from './currency-search'

const item = (code: string, name: string, symbol = code): CurrencyCatalogItem => ({
    code,
    name,
    symbol,
    decimals: 2,
    rate: null,
})

const catalog = [
    item('AUD', 'Australian Dollar', 'AU$'),
    item('EUR', 'Euro', '€'),
    item('GBP', 'British Pound', '£'),
    item('JPY', 'Japanese Yen', '¥'),
    item('RUB', 'Russian Ruble', '₽'),
    item('USD', 'US Dollar', '$'),
]
const codes = (list: CurrencyCatalogItem[]) => list.map((c) => c.code)

describe('searchCurrencies', () => {
    it('puts the popular ones first, then the rest in catalog order', () => {
        const { shown } = searchCurrencies(catalog, '', { popular: ['USD', 'EUR', 'GBP'] })
        expect(codes(shown)).toEqual(['USD', 'EUR', 'GBP', 'AUD', 'JPY', 'RUB'])
    })

    it('finds Japanese Yen by its name, code or symbol', () => {
        for (const query of ['yen', 'JPY', 'jp', '¥']) {
            expect(codes(searchCurrencies(catalog, query).shown), query).toContain('JPY')
        }
        expect(codes(searchCurrencies(catalog, 'yen').shown)).toEqual(['JPY'])
    })

    it('ranks a code that starts with the query before a name that contains it', () => {
        const list = [item('XEU', 'Something Eur'), item('EUR', 'Euro')]
        expect(codes(searchCurrencies(list, 'eur').shown)).toEqual(['EUR', 'XEU'])
    })

    it('leaves out the codes already chosen', () => {
        const { shown } = searchCurrencies(catalog, '', { exclude: new Set(['USD', 'EUR']) })
        expect(codes(shown)).not.toContain('USD')
        expect(codes(shown)).not.toContain('EUR')
    })

    it('caps the list and says how many matched', () => {
        const many = Array.from({ length: 170 }, (_, i) => item(`C${String(i).padStart(3, '0')}`, `Currency ${i}`))
        const { shown, total } = searchCurrencies(many, '')
        expect(shown).toHaveLength(MAX_CURRENCY_OPTIONS)
        expect(total).toBe(170)
    })
})

describe('manualCode', () => {
    it('offers a typed ISO-like code that the list does not have', () => {
        expect(manualCode(catalog, ' sek ')).toBe('SEK')
    })

    it('offers nothing for a known code, a non-code or an empty query', () => {
        expect(manualCode(catalog, 'usd')).toBeNull()
        expect(manualCode(catalog, 'US Dollar')).toBeNull()
        expect(manualCode(catalog, 'x')).toBeNull()
        expect(manualCode(catalog, '')).toBeNull()
    })
})
