import { describe, expect, it, vi } from 'vitest'

import { formatBytes, formatDateTime } from './format'

vi.mock('@/lib/i18n', () => ({ intlLocale: () => 'en-US' }))

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

    it('formats a timestamp for the current locale', () => {
        expect(formatDateTime('2026-03-05T14:30:00')).toBe(new Date('2026-03-05T14:30:00').toLocaleString('en-US'))
    })
})
