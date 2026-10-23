import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { raw } from '@/test/api-mocks'

import PasskeySettingsPage from './passkeys'

const meta = {
    title: 'pages/settings/Passkeys',
    component: PasskeySettingsPage,
    parameters: { a11y: overlayA11y, route: '/settings/passkeys' },
    beforeEach: ({ msw }) =>
        msw.use(
            raw('get', '/auth/webauthn/credentials', {
                credentials: [{ id: 1, name: 'Laptop', aaguid: null, lastUsedAt: '2030-01-05T10:00:00Z', createdAt: '2030-01-01T00:00:00Z' }],
            }),
            raw('get', '/auth/2fa/status', { enabled: false, pending_confirmation: false, recovery_codes_remaining: null }),
        ),
} satisfies Meta<typeof PasskeySettingsPage>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Laptop')).toBeVisible()
        // The tabs of the user's own settings.
        await expect(canvas.getByRole('link', { name: /security/i })).toHaveAttribute('href', '/settings/security')
    },
}
export const InRussian: Story = { tags: ['mobile-only'], globals: { locale: 'ru' } }
export const Empty: Story = {
    beforeEach: ({ msw }) => msw.use(raw('get', '/auth/webauthn/credentials', { credentials: [] })),
}
