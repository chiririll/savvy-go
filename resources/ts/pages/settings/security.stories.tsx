import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { hang, raw } from '@/test/api-mocks'

import SecuritySettingsPage from './security'

const status = (enabled: boolean, left: number | null = null) =>
    raw('get', '/auth/2fa/status', { enabled, pending_confirmation: false, recovery_codes_remaining: left })

const meta = {
    title: 'pages/settings/Security',
    component: SecuritySettingsPage,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) => msw.use(status(false)),
} satisfies Meta<typeof SecuritySettingsPage>

export default meta
type Story = StoryObj<typeof meta>

export const TwoFactorOff: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByRole('switch')).not.toBeChecked()
        await expect(canvas.getByRole('button', { name: /sign out other/i })).toBeVisible()
    },
}
export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/auth/2fa/status')) }

export const TwoFactorOn: Story = {
    beforeEach: ({ msw }) => msw.use(status(true, 8)),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('switch')).toBeChecked()
        await expect(await within(canvasElement).findByRole('button', { name: /regenerate/i })).toBeVisible()
    },
}
