// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

vi.mock('react-i18next', async (importActual) => ({
    ...(await importActual<typeof import('react-i18next')>()),
    useTranslation: () => ({ t: (k: string) => k }),
}))
vi.mock('@/hooks/use-theme', () => ({ useTheme: () => ({ theme: 'light' }) }))
vi.mock('emoji-picker-react', () => ({
    default: () => null,
    EmojiStyle: { NATIVE: 'native' },
    Theme: { DARK: 'dark', LIGHT: 'light' },
}))

import { EmojiPicker } from './EmojiPicker'

afterEach(() => cleanup())

describe('EmojiPicker', () => {
    it('shows a placeholder, not a real emoji, when empty', () => {
        render(<EmojiPicker value="" onChange={() => {}} />)
        expect(screen.queryByText('😀')).toBeNull()
        expect(screen.getByTestId('emoji-placeholder')).toBeTruthy()
    })

    it('shows the chosen emoji', () => {
        render(<EmojiPicker value="🏠" onChange={() => {}} />)
        expect(screen.getByText('🏠')).toBeTruthy()
        expect(screen.queryByTestId('emoji-placeholder')).toBeNull()
    })
})
