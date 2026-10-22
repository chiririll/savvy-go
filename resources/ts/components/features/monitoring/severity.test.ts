import { describe, expect, it } from 'vitest'

import { formatBytes } from './format'
import { severityFor } from './severity'

describe('severityFor', () => {
    it('treats unknown usage as ok', () => {
        expect(severityFor(null)).toBe('ok')
    })

    it('switches to warn at 75 and crit at 90', () => {
        expect(severityFor(74.9)).toBe('ok')
        expect(severityFor(75)).toBe('warn')
        expect(severityFor(89.9)).toBe('warn')
        expect(severityFor(90)).toBe('crit')
    })
})

describe('formatBytes', () => {
    it('shows a dash for missing values and zero as bytes', () => {
        expect(formatBytes(null)).toBe('—')
        expect(formatBytes(undefined)).toBe('—')
        expect(formatBytes(0)).toBe('0 B')
    })

    it('scales to the right unit', () => {
        expect(formatBytes(512)).toBe('512 B')
        expect(formatBytes(1536)).toBe('1.5 KB')
        expect(formatBytes(1024 ** 3)).toBe('1 GB')
    })
})
