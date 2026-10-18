import { describe, expect, it } from 'vitest'

import type { SpacePresets } from '@/types/spaces'
import {
    addInvitation,
    buildContent,
    canRemoveCurrency,
    draftProblems,
    effectiveSourceId,
    effectiveStep,
    emptyDraft,
    nextStep,
    noCopy,
    previousStep,
    removeCategory,
    removeInvitation,
    toCopySource,
    toRequest,
    type CopySource,
} from './draft'

const presets: SpacePresets = {
    categories: [
        { name: '#HOUSING', type: 'expense', icon: 'home', color: '#111111', isDefault: false },
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

    it('goes through the copy step when there is a space to copy', () => {
        expect(nextStep('name', 2)).toBe('copy')
        expect(nextStep('copy', 2)).toBe('entities')
        expect(previousStep('entities', 2)).toBe('copy')
    })

    it('goes from entities straight to the summary, as inviting is no step', () => {
        expect(nextStep('entities', 0)).toBe('confirm')
        expect(previousStep('confirm', 0)).toBe('entities')
    })

    it('stays inside the first and last step', () => {
        expect(previousStep('name', 1)).toBe('name')
        expect(nextStep('confirm', 1)).toBe('confirm')
    })
})

describe('effective step and source', () => {
    it('returns to the name step when there is nothing to copy from', () => {
        expect(effectiveStep('copy', 0)).toBe('name')
        expect(effectiveStep('copy', 2)).toBe('copy')
        expect(effectiveStep('entities', 0)).toBe('entities')
    })

    it('ignores a source space that is gone', () => {
        expect(effectiveSourceId(3, [1, 3])).toBe(3)
        expect(effectiveSourceId(3, [1, 2])).toBeNull()
        expect(effectiveSourceId(null, [1, 2])).toBeNull()
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
        expect(content.categories.map((c) => c.name)).toContain('#HOUSING')
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
        expect(content.categories.filter((c) => c.type === 'expense' && c.isDefault).map((c) => c.name)).toEqual([
            'Misc',
        ])
        expect(content.categories.filter((c) => c.type === 'income' && c.isDefault).map((c) => c.name)).toEqual([
            '#OTHER_INCOME',
        ])
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
        const rent = categories.findIndex((c) => c.name === '#HOUSING')
        expect(removeCategory(categories, rent).map((c) => c.name)).not.toContain('#HOUSING')
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

    it('adds the account currency to the list and drops duplicate tags', () => {
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

describe('invitation list', () => {
    it('adds an email once, whatever its case', () => {
        let list = addInvitation([], { email: ' Bob@Example.org ', role: 'editor' })
        expect(list).toHaveLength(1)
        expect(list[0]).toMatchObject({ email: 'Bob@Example.org', role: 'editor' })
        list = addInvitation(list, { email: 'bob@example.org', role: 'viewer' })
        expect(list).toHaveLength(1)
    })

    it('does not add something that is not an address', () => {
        expect(addInvitation([], { email: 'not-an-address', role: 'viewer' })).toEqual([])
    })

    it('adds a link invitation without an address, as many as asked', () => {
        let list = addInvitation([], { email: null, role: 'viewer' })
        list = addInvitation(list, { email: null, role: 'viewer' })
        expect(list).toHaveLength(2)
        expect(list[0].key).not.toBe(list[1].key)
    })

    it('keeps the link invitation note, trimmed, and drops an empty one', () => {
        const list = addInvitation([], { email: null, note: '  Kim from work ', role: 'viewer' })
        expect(list[0].note).toBe('Kim from work')
        expect(addInvitation([], { email: null, note: '   ', role: 'viewer' })[0].note).toBeNull()
    })

    it('has no note on an invitation by email', () => {
        expect(addInvitation([], { email: 'a@example.org', note: 'ignored', role: 'viewer' })[0].note).toBeNull()
    })

    it('removes one entry by its key', () => {
        const list = addInvitation(addInvitation([], { email: 'a@example.org', role: 'viewer' }), {
            email: 'b@example.org',
            role: 'viewer',
        })
        expect(removeInvitation(list, list[0].key).map((i) => i.email)).toEqual(['b@example.org'])
    })

    it('is not part of the draft or the create-space request', () => {
        expect(emptyDraft()).not.toHaveProperty('invitations')
        expect(toRequest({ ...emptyDraft(), name: 'Home', baseCurrency: 'EUR' })).not.toHaveProperty('invitations')
    })
})

describe('look in the request', () => {
    it('sends the emoji and color only when picked', () => {
        const draft = { ...emptyDraft(), name: ' Trip ', baseCurrency: 'EUR' }
        expect(toRequest(draft)).not.toHaveProperty('emoji')
        expect(toRequest(draft)).not.toHaveProperty('color')
        const body = toRequest({ ...draft, emoji: '🏖️', color: '#ff8800' })
        expect(body.emoji).toBe('🏖️')
        expect(body.color).toBe('#ff8800')
    })
})
