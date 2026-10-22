import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { UserSettingsTabs } from './UserSettingsTabs'

const meta = {
    title: 'features/settings/UserSettingsTabs',
    component: UserSettingsTabs,
    parameters: { route: '/settings/security' },
} satisfies Meta<typeof UserSettingsTabs>

export default meta
type Story = StoryObj<typeof meta>

export const Security: Story = {
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('link', { name: /security/i })).toHaveAttribute('aria-current', 'page')
    },
}

export const Api: Story = {
    parameters: { route: '/settings/api' },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('link', { name: /api/i })).toHaveAttribute('aria-current', 'page')
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }
