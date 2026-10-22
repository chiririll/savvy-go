import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, within } from 'storybook/test'

import i18n from '@/lib/i18n'
import { overlayA11y } from '@/test/a11y'

import { LanguageSwitcher } from './LanguageSwitcher'

const meta = {
    title: 'shared/appearance/LanguageSwitcher',
    component: LanguageSwitcher,
    parameters: { a11y: overlayA11y },
    afterEach: async () => {
        await i18n.changeLanguage('en')
    },
} satisfies Meta<typeof LanguageSwitcher>

export default meta
type Story = StoryObj<typeof meta>

export const Compact: Story = {}
export const Select: Story = { args: { variant: 'select' } }
export const Auth: Story = { args: { variant: 'auth' } }
export const InRussian: Story = { globals: { locale: 'ru' } }

export const CompactMenu: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button'))
        await expect(await screen.findByRole('menuitem', { name: /English/ })).toBeVisible()
        await expect(screen.getByRole('menuitem', { name: /Русский/ })).toBeVisible()
    },
}

export const ChangesLanguage: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button'))
        await userEvent.click(await screen.findByRole('menuitem', { name: /Русский/ }))
        await expect(i18n.resolvedLanguage).toBe('ru')
    },
}
