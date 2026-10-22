import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'

import { ThemeSwitcher } from './ThemeSwitcher'

const meta = {
    title: 'shared/appearance/ThemeSwitcher',
    component: ThemeSwitcher,
    parameters: { a11y: overlayA11y },
    afterEach: () => {
        localStorage.removeItem('theme')
    },
} satisfies Meta<typeof ThemeSwitcher>

export default meta
type Story = StoryObj<typeof meta>

export const Icon: Story = {}
export const Select: Story = { args: { variant: 'select' } }

export const MenuOptions: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button'))
        await expect(await screen.findByRole('menuitemradio', { name: /light/i })).toBeVisible()
        await expect(screen.getByRole('menuitemradio', { name: /dark/i })).toBeVisible()
        await expect(screen.getByRole('menuitemradio', { name: /auto/i })).toBeVisible()
    },
}

export const ChoosesDark: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button'))
        await userEvent.click(await screen.findByRole('menuitemradio', { name: /dark/i }))
        await expect(localStorage.getItem('theme')).toBe('dark')
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }
