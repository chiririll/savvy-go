// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'

import { ColorIcon } from './ColorIcon'

afterEach(() => cleanup())

describe('ColorIcon', () => {
    it('draws the emoji on a tint of the colour, not the solid colour', () => {
        render(<ColorIcon emoji="🏖️" color="#ff8800" />)
        const el = screen.getByText('🏖️')
        expect(el.style.backgroundColor).toBe('rgba(255, 136, 0, 0.2)')
        expect(el.className).toContain('text-foreground')
        expect(el.className).not.toContain('text-white')
    })

    it('falls back to the given text without an emoji', () => {
        render(<ColorIcon emoji={null} color="#112233" fallback="T" />)
        expect(screen.getByText('T')).toBeTruthy()
    })

    it('uses the theme colours without a colour', () => {
        render(<ColorIcon emoji="🏠" color={null} />)
        const el = screen.getByText('🏠')
        expect(el.style.backgroundColor).toBe('')
        expect(el.className).toContain('bg-muted')
    })

    it('accepts an empty colour string', () => {
        render(<ColorIcon emoji="🏠" color="" />)
        expect(screen.getByText('🏠').style.backgroundColor).toBe('')
    })
})
