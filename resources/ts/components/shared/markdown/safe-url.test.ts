import { describe, expect, it } from 'vitest'

import { safeUrl } from './safe-url'

describe('safeUrl', () => {
    it.each([
        ['https://example.com/a?b=c:d', 'https://example.com/a?b=c:d'],
        ['HTTP://example.com', 'HTTP://example.com'],
        ['mailto:a@b.c', 'mailto:a@b.c'],
        ['tel:+123', 'tel:+123'],
        ['/relative/path', '/relative/path'],
        ['page.html', 'page.html'],
        ['#anchor:x', '#anchor:x'],
        ['javascript:alert(1)', ''],
        ['JaVaScRiPt:alert(1)', ''],
        [' java\tscript:alert(1)', ''],
        ['data:text/html;base64,AAAA', ''],
        ['vbscript:x', ''],
    ])('%s', (input, want) => {
        expect(safeUrl(input)).toBe(want)
    })
})
