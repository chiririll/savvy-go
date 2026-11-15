import { describe, expect, it } from 'vitest'

import { colorShades } from './color-shades'

const lightness = (hex: string) => {
    const [r, g, b] = [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16) / 255)
    return (Math.max(r, g, b) + Math.min(r, g, b)) / 2
}

describe('colorShades', () => {
    it('starts at the colour itself and gives each next one a distinct shade', () => {
        const shades = colorShades('#2563eb', 4)
        expect(shades).toHaveLength(4)
        expect(shades[0]).toBe('#2563eb')
        expect(new Set(shades).size).toBe(4)
        shades.forEach((s) => expect(s).toMatch(/^#[0-9a-f]{6}$/))
    })

    it('keeps shades readable: neither near white nor near black', () => {
        for (const base of ['#fafafa', '#0a0a0a', '#16a34a']) {
            colorShades(base, 6).slice(1).forEach((s) => {
                expect(lightness(s)).toBeGreaterThan(0.2)
                expect(lightness(s)).toBeLessThan(0.85)
            })
        }
    })

    it('reads the short form and leaves a colour it cannot read as it is', () => {
        expect(colorShades('#abc', 1)).toEqual(['#aabbcc'])
        expect(colorShades('tomato', 2)).toEqual(['tomato', 'tomato'])
        expect(colorShades('#2563eb', 0)).toEqual([])
    })
})
