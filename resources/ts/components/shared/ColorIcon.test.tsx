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

    it('shows the icon node without an emoji, and the emoji over the icon', () => {
        const { rerender } = render(<ColorIcon icon={<svg data-testid="glyph" />} fallback="T" />)
        expect(screen.getByTestId('glyph')).toBeTruthy()
        expect(screen.queryByText('T')).toBeNull()
        rerender(<ColorIcon emoji="🏠" icon={<svg data-testid="glyph" />} />)
        expect(screen.queryByTestId('glyph')).toBeNull()
        expect(screen.getByText('🏠')).toBeTruthy()
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

    it('uses the tone, not the default, without a colour, and ignores it with one', () => {
        const { rerender } = render(<ColorIcon emoji="🏠" tone="bg-info/15 text-info" />)
        expect(screen.getByText('🏠').className).toContain('bg-info/15')
        expect(screen.getByText('🏠').className).not.toContain('bg-muted')
        rerender(<ColorIcon emoji="🏠" color="#ff8800" tone="bg-info/15 text-info" />)
        expect(screen.getByText('🏠').className).not.toContain('bg-info/15')
    })

    it('is a rounded square of 32px by default', () => {
        render(<ColorIcon emoji="🏠" />)
        const el = screen.getByText('🏠')
        expect(el.className).toContain('size-8')
        expect(el.className).toContain('rounded-lg')
    })

    it('can be a 40px circle', () => {
        render(<ColorIcon emoji="🏠" size="lg" shape="circle" />)
        const el = screen.getByText('🏠')
        expect(el.className).toContain('size-10')
        expect(el.className).toContain('rounded-full')
        expect(el.className).not.toContain('rounded-lg')
    })
})
