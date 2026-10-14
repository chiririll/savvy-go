import { describe, expect, it } from 'vitest'

import type { SpacePresets } from '@/types/spaces'
import {
    buildContent,
    canRemoveCurrency,
    draftProblems,
    emptyDraft,
    nextStep,
    noCopy,
    previousStep,
    removeCategory,
    toCopySource,
    toRequest,
    type CopySource,
} from './draft'

const presets: SpacePresets = {
    categories: [
        { name: '#RENT', type: 'expense', icon: 'home', color: '#111111', isDefault: false },
        { name: '#OTHER', type: 'expense', icon: 'tag', color: '#222222', isDefault: true },
        { name: '#SALARY', type: 'income', icon: 'wallet', color: '#333333', isDefault: false },
        { name: '#OTHER_INCOME', type: 'income', icon: 'tag', color: '#444444', isDefault: true },
    ],
    tags: ['Family', 'Work'],
}

const source: CopySource = {
    baseCurrency: 'EUR',
    currencies: ['EUR', 'USD'],
    accounts: [
        { name: 'Card', type: 'bank', currency: 'EUR' },
        { name: 'Cash', type: 'cash', currency: 'GBP' },
    ],
    categories: [
        { name: 'Food', type: 'expense', icon: 'x', color: '#000000', isDefault: false },
        { name: 'Misc', type: 'expense', icon: 'x', color: '#000000', isDefault: true },
    ],
    tags: ['Mine'],
}

describe('steps', () => {
    it('skips the copy step when the user belongs to no other space', () => {
        expect(nextStep('name', 0)).toBe('entities')
        expect(previousStep('entities', 0)).toBe('name')
    })

    it('goes through the copy step when there is a space to copy from', () => {
        expect(nextStep('name', 2)).toBe('copy')
        expect(nextStep('copy', 2)).toBe('entities')
        expect(previousStep('entities', 2)).toBe('copy')
    })

    it('stays inside the first and last step', () => {
        expect(previousStep('intro', 1)).toBe('intro')
        expect(nextStep('confirm', 1)).toBe('confirm')
    })
})

describe('buildContent', () => {
    it('starts from the defaults when nothing is copied', () => {
        const content = buildContent(noCopy(), source, presets)
        expect(content.baseCurrency).toBeNull()
        expect(content.currencies).toEqual([])
        expect(content.accounts).toEqual([])
        expect(content.categories).toHaveLength(4)
        expect(content.tags).toEqual(['Family', 'Work'])
    })

    it('uses the defaults without a source', () => {
        const content = buildContent({ currencies: true, accounts: true, categories: true, tags: true }, null, presets)
        expect(content.accounts).toEqual([])
        expect(content.tags).toEqual(['Family', 'Work'])
    })

    it('copies only the ticked kinds', () => {
        const content = buildContent({ ...noCopy(), tags: true, currencies: true }, source, presets)
        expect(content.tags).toEqual(['Mine'])
        expect(content.baseCurrency).toBe('EUR')
        expect(content.currencies).toEqual(['USD'])
        expect(content.accounts).toEqual([])
        expect(content.categories.map((c) => c.name)).toContain('#RENT')
    })

    it('copies accounts without balances', () => {
        const content = buildContent({ ...noCopy(), accounts: true }, source, presets)
        expect(content.accounts.map((a) => [a.name, a.type, a.currency, a.initialBalance])).toEqual([
            ['Card', 'bank', 'EUR', ''],
            ['Cash', 'cash', 'GBP', ''],
        ])
    })

    it('pulls in the currency of a copied account', () => {
        const content = buildContent({ ...noCopy(), accounts: true }, source, presets)
        expect(content.currencies).toEqual(['EUR', 'GBP'])
    })

    it('pulls in a missing currency even when currencies are copied', () => {
        const content = buildContent({ ...noCopy(), accounts: true, currencies: true }, source, presets)
        expect(content.baseCurrency).toBe('EUR')
        expect(content.currencies).toEqual(['USD', 'GBP'])
    })

    it('keeps a default category per type when copied categories lack one', () => {
        const content = buildContent({ ...noCopy(), categories: true }, source, presets)
        expect(content.categories.filter((c) => c.type === 'expense' && c.isDefault).map((c) => c.name)).toEqual(['Misc'])
        expect(content.categories.filter((c) => c.type === 'income' && c.isDefault).map((c) => c.name)).toEqual(['#OTHER_INCOME'])
    })
})

describe('categories', () => {
    it('cannot remove the type defaults', () => {
        const categories = buildContent(noCopy(), null, presets).categories
        const other = categories.findIndex((c) => c.name === '#OTHER')
        const otherIncome = categories.findIndex((c) => c.name === '#OTHER_INCOME')
        expect(removeCategory(categories, other)).toBe(categories)
        expect(removeCategory(categories, otherIncome)).toBe(categories)
    })

    it('removes any other category', () => {
        const categories = buildContent(noCopy(), null, presets).categories
        const rent = categories.findIndex((c) => c.name === '#RENT')
        expect(removeCategory(categories, rent).map((c) => c.name)).not.toContain('#RENT')
    })
})

describe('currencies', () => {
    it('keeps the base and the currencies accounts use', () => {
        const content = buildContent({ ...noCopy(), accounts: true, currencies: true }, source, presets)
        expect(canRemoveCurrency(content, 'EUR')).toBe(false)
        expect(canRemoveCurrency(content, 'GBP')).toBe(false)
        expect(canRemoveCurrency(content, 'USD')).toBe(true)
    })
})

describe('request', () => {
    const ready = () => ({
        ...emptyDraft(),
        ...buildContent({ ...noCopy(), accounts: true }, source, presets),
        name: ' Home ',
        baseCurrency: 'EUR',
    })

    it('reports what is missing', () => {
        expect(draftProblems(emptyDraft())).toEqual(['name', 'baseCurrency'])
        expect(draftProblems(ready())).toEqual([])
    })

    it('rejects an unnamed account and a malformed balance', () => {
        const draft = ready()
        draft.accounts[0].name = ' '
        draft.accounts[1].initialBalance = '1e3'
        expect(draftProblems(draft)).toEqual(['accounts', 'balances'])
    })

    it('sends balances as decimal strings and leaves empty ones out', () => {
        const draft = ready()
        draft.accounts[0].initialBalance = '10,50'
        const body = toRequest(draft)
        expect(body.name).toBe('Home')
        expect(body.accounts?.[0]).toEqual({ name: 'Card', type: 'bank', currency: 'EUR', initialBalance: '10.50' })
        expect(body.accounts?.[1]).not.toHaveProperty('initialBalance')
    })

    it('adds the currency of an account to the list and drops duplicate tags', () => {
        const draft = { ...ready(), currencies: [], tags: ['Family', ' family ', '', 'Work'] }
        const body = toRequest(draft)
        expect(body.currencies).toEqual(['GBP'])
        expect(body.tags).toEqual(['Family', 'Work'])
    })
})

describe('toCopySource', () => {
    it('reads the base, leaves out debts and accounts without a currency', () => {
        const copy = toCopySource({
            currencies: [
                { code: 'USD', isBase: false },
                { code: 'EUR', isBase: true },
            ],
            accounts: [
                { name: 'Card', type: 'bank', currency: { code: 'EUR' } },
                { name: 'Loan', type: 'debt', currency: { code: 'EUR' } },
                { name: 'Lost', type: 'cash', currency: null },
            ],
            categories: [{ name: 'Food', type: 'expense', icon: null, color: null, isDefault: false }],
            tags: [{ name: 'Mine' }],
        })
        expect(copy.baseCurrency).toBe('EUR')
        expect(copy.currencies).toEqual(['USD', 'EUR'])
        expect(copy.accounts).toEqual([{ name: 'Card', type: 'bank', currency: 'EUR' }])
        expect(copy.categories[0]).toMatchObject({ icon: '', color: '' })
        expect(copy.tags).toEqual(['Mine'])
    })
})
