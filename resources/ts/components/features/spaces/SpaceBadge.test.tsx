// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'

import { SpaceBadge } from './SpaceBadge'

afterEach(() => cleanup())

describe('SpaceBadge', () => {
    it('shows the emoji on its color', () => {
        render(<SpaceBadge space={{ name: 'Trip', emoji: '🏖️', color: '#ff8800' }} />)
        expect(screen.getByText('🏖️').style.backgroundColor).toBe('rgb(255, 136, 0)')
    })

    it('falls back to the first letter of the name', () => {
        render(<SpaceBadge space={{ name: 'trip home', emoji: null, color: '#112233' }} />)
        expect(screen.getByText('T').style.backgroundColor).toBe('rgb(17, 34, 51)')
    })

    it('uses the theme color when none is set', () => {
        render(<SpaceBadge space={{ name: 'Home', emoji: null, color: null }} />)
        expect(screen.getByText('H').style.backgroundColor).toBe('')
    })
})
