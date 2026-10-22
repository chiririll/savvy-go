import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { SpaceBadge } from './SpaceBadge'

const meta = {
    title: 'features/spaces/SpaceBadge',
    component: SpaceBadge,
    parameters: { layout: 'padded' },
    args: { space: { name: 'Home', emoji: '🏠', color: '#2563eb' } },
} satisfies Meta<typeof SpaceBadge>

export default meta
type Story = StoryObj<typeof meta>

export const WithEmoji: Story = {}

export const FirstLetterWithoutEmoji: Story = {
    args: { space: { name: '  family', emoji: null, color: '#16a34a' } },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('F')).toBeVisible()
    },
}

// No color falls back to the theme's primary color.
export const WithoutColor: Story = { args: { space: { name: 'Work', emoji: null, color: null } } }
export const WithoutColorInDarkTheme: Story = { ...WithoutColor, globals: { theme: 'dark' } }

export const BlankName: Story = {
    args: { space: { name: ' ', emoji: null, color: null } },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('?')).toBeVisible()
    },
}

export const Large: Story = { args: { className: 'size-16 text-2xl' } }
