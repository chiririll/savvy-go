import { describe, expect, it, vi } from 'vitest'

import { formatBytes, formatDateTime } from './format'

const app = vi.hoisted(() => ({ locale: 'en-US' }))
vi.mock('@/lib/i18n', () => ({ intlLocale: () => app.locale }))

describe('formatBytes', () => {
    it('handles zero and empty values', () => {
        expect(formatBytes(0)).toBe('0 B')
        expect(formatBytes(NaN)).toBe('0 B')
    })

    it('scales by 1024 and trims a trailing .0', () => {
        expect(formatBytes(1)).toBe('1 B')
        expect(formatBytes(1023)).toBe('1023 B')
        expect(formatBytes(1024)).toBe('1 KB')
        expect(formatBytes(1536)).toBe('1.5 KB')
        expect(formatBytes(1024 * 1024)).toBe('1 MB')
        expect(formatBytes(5 * 1024 ** 3)).toBe('5 GB')
    })

    it('does not run past the largest unit', () => {
        expect(formatBytes(2048 * 1024 ** 4)).toBe('2048 TB')
    })
})

describe('formatDateTime', () => {
    it('returns an empty string for missing values', () => {
        expect(formatDateTime(null)).toBe('')
        expect(formatDateTime(undefined)).toBe('')
        expect(formatDateTime('')).toBe('')
    })

    // Two locales, so at least one differs from the host's default and the test
    // fails if the app locale is ignored. No offset in the input, so it is local
    // time wherever the test runs.
    it.each([
        ['ru-RU', '05.03.2026, 14:30:00'],
        ['en-US', '3/5/2026, 2:30:00 PM'],
    ])('formats a timestamp for the app locale (%s)', (locale, expected) => {
        app.locale = locale
        expect(formatDateTime('2026-03-05T14:30:00')).toBe(expected)
    })
})
