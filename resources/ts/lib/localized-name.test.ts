import { describe, expect, it, vi } from 'vitest'

import { defaultNameKey, localizeDefaultName, toStoredDefaultName } from './localized-name'

// Only SALARY and FOOD have a translation.
vi.mock('@/lib/i18n', () => {
    const names: Record<string, string> = { 'categories.SALARY': 'Зарплата', 'categories.FOOD': 'Еда' }
    return {
        default: {
            exists: (key: string) => key in names,
            t: (key: string) => names[key] ?? key,
        },
    }
})

describe('defaultNameKey', () => {
    it('extracts the key of a stored default name', () => {
        expect(defaultNameKey('#SALARY')).toBe('SALARY')
        expect(defaultNameKey('  #FOOD_2  ')).toBe('FOOD_2')
    })

    it('rejects anything that is not a default name', () => {
        expect(defaultNameKey('Salary')).toBeNull()
        expect(defaultNameKey('#salary')).toBeNull()
        expect(defaultNameKey('#')).toBeNull()
        expect(defaultNameKey('# SALARY')).toBeNull()
        expect(defaultNameKey('#1ABC')).toBeNull()
        expect(defaultNameKey('')).toBeNull()
        expect(defaultNameKey(null)).toBeNull()
        expect(defaultNameKey(undefined)).toBeNull()
    })
})

describe('localizeDefaultName', () => {
    it('translates a default name that has a translation', () => {
        expect(localizeDefaultName('#SALARY')).toBe('Зарплата')
    })

    it('keeps a default name that has no translation', () => {
        expect(localizeDefaultName('#UNKNOWN')).toBe('#UNKNOWN')
    })

    it('keeps user-chosen names as they are', () => {
        expect(localizeDefaultName('My category')).toBe('My category')
    })

    it('returns an empty string for no name', () => {
        expect(localizeDefaultName(null)).toBe('')
        expect(localizeDefaultName(undefined)).toBe('')
        expect(localizeDefaultName('')).toBe('')
    })
})

describe('toStoredDefaultName', () => {
    it('stores the original default name when the user left the translation untouched', () => {
        expect(toStoredDefaultName('Зарплата', '#SALARY')).toBe('#SALARY')
        expect(toStoredDefaultName('  Зарплата ', ' #SALARY ')).toBe('#SALARY')
    })

    it('stores what the user typed once they change it', () => {
        expect(toStoredDefaultName('Оклад', '#SALARY')).toBe('Оклад')
    })

    it('trims plain input', () => {
        expect(toStoredDefaultName('  Food  ')).toBe('Food')
    })

    it('does not reuse the original when it was a custom name', () => {
        expect(toStoredDefaultName('Mine', 'Mine')).toBe('Mine')
    })
})
