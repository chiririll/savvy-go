import { describe, expect, it, vi } from 'vitest'

import { getApiErrorMessage, resolveUserFacingText } from './api-error'

// A tiny translation table per namespace; `t` echoes the key for a miss, like i18next.
vi.mock('@/lib/i18n', () => {
    const dict: Record<string, Record<string, string>> = {
        common: { 'validation.required': 'This field is required', 'validation.amountTooLarge': 'Amount is too large' },
        forms: { 'transactions.accountRequired': 'Pick an account', 'debts.debtNotFound': 'Debt not found' },
        pages: { 'accounts.title': 'Accounts' },
        auth: { 'login.failed': 'Wrong email or password' },
        settings: {},
        nav: {},
    }
    return {
        default: {
            exists: (key: string, { ns }: { ns: string }) => key in (dict[ns] ?? {}),
            t: (key: string, { ns }: { ns: string }) => dict[ns]?.[key] ?? key,
        },
    }
})

describe('resolveUserFacingText', () => {
    it('leaves ordinary sentences alone', () => {
        expect(resolveUserFacingText('Something went wrong.')).toBe('Something went wrong.')
        expect(resolveUserFacingText('Email is taken')).toBe('Email is taken')
    })

    it('translates a key that exists in some namespace', () => {
        expect(resolveUserFacingText('login.failed')).toBe('Wrong email or password')
        expect(resolveUserFacingText('accounts.title')).toBe('Accounts')
    })

    it('understands a namespace prefix', () => {
        expect(resolveUserFacingText('auth.login.failed')).toBe('Wrong email or password')
        expect(resolveUserFacingText('common.validation.required')).toBe('This field is required')
    })

    it('finds validation messages by the last part of the key', () => {
        expect(resolveUserFacingText('amount.required')).toBe('This field is required')
        expect(resolveUserFacingText('amount.amount_too_large')).toBe('Amount is too large')
    })

    it('finds transaction and debt messages in the forms namespace', () => {
        expect(resolveUserFacingText('account_required')).toBe('Pick an account')
        expect(resolveUserFacingText('debt_not_found')).toBe('Debt not found')
    })

    it('shows an unknown key as it is', () => {
        expect(resolveUserFacingText('totally.unknown.key')).toBe('totally.unknown.key')
        expect(resolveUserFacingText('Forbidden')).toBe('Forbidden')
    })
})

describe('getApiErrorMessage', () => {
    it('falls back when there is nothing usable', () => {
        expect(getApiErrorMessage(undefined, 'Fallback')).toBe('Fallback')
        expect(getApiErrorMessage(null, 'Fallback')).toBe('Fallback')
        expect(getApiErrorMessage('text', 'Fallback')).toBe('Fallback')
        expect(getApiErrorMessage({}, 'Fallback')).toBe('Fallback')
        expect(getApiErrorMessage({ message: '' }, 'Fallback')).toBe('Fallback')
    })

    it('uses the message', () => {
        expect(getApiErrorMessage({ message: 'Boom!' }, 'Fallback')).toBe('Boom!')
    })

    it('prefers the first non-empty detail over the message', () => {
        const error = { message: 'The given data was invalid.', details: { name: ['', 'validation.required'], other: ['x'] } }
        expect(getApiErrorMessage(error, 'Fallback')).toBe('This field is required')
    })

    it('ignores details without a usable string', () => {
        expect(getApiErrorMessage({ message: 'Boom!', details: { name: [] } }, 'Fallback')).toBe('Boom!')
        expect(getApiErrorMessage({ message: 'Boom!', details: {} }, 'Fallback')).toBe('Boom!')
    })

    it('translates a message that is a key', () => {
        expect(getApiErrorMessage({ message: 'login.failed' }, 'Fallback')).toBe('Wrong email or password')
    })
})
