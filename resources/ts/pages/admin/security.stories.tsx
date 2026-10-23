import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { reply } from '@/test/api-mocks'

import AdminSecurityPage from './security'

const meta = {
    title: 'pages/admin/Security',
    component: AdminSecurityPage,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) =>
        msw.use(
            reply('get', '/admin/oauth/registration', { open: false, expires_at: null }),
            reply('get', '/admin/oauth/clients', []),
            reply('get', '/admin/keys', {
                own: { kid: 'k-own', publicKey: 'MCowBQYDK2VwAyEAown' },
                trusted: [{ kid: 'k-friend', publicKey: 'MCowBQYDK2VwAyEAfriend', name: 'Parents server', createdAt: null }],
            }),
        ),
} satisfies Meta<typeof AdminSecurityPage>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Parents server')).toBeVisible()
        await expect(canvas.getByRole('heading', { level: 1 })).toBeVisible()
    },
}
export const InRussian: Story = { globals: { locale: 'ru' } }
