import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { reply } from '@/test/api-mocks'
import { space } from '@/test/app-state'

import SpaceSettingsPage from './space'

const members = [
    { userId: 1, name: 'Kirill', email: 'kirill@example.com', role: 'admin', createdAt: null },
    { userId: 2, name: 'Alex', email: 'alex@example.com', role: 'editor', createdAt: null },
]

const meta = {
    title: 'pages/settings/Space',
    component: SpaceSettingsPage,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) =>
        msw.use(
            reply('get', '/spaces/1', { ...space(), quota: 0, size: 3_500_000 }),
            reply('get', '/members', members),
            reply('get', '/invitations', []),
            reply('get', '/invitation-methods', { email: true, link: true }),
            reply('get', '/links', []),
            reply('get', '/sync-group', null),
            reply('get', '/transfers', []),
            reply('get', '/audit', []),
        ),
} satisfies Meta<typeof SpaceSettingsPage>

export default meta
type Story = StoryObj<typeof meta>

export const AsAdmin: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByRole('heading', { level: 1 })).toHaveTextContent('Home')
        await expect(await canvas.findByText('Alex')).toBeVisible()
    },
}

// A member who is not an admin gets neither the invitations nor the audit log.
export const AsViewer: Story = {
    parameters: { app: { role: 'viewer' } },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Alex')).toBeVisible()
    },
}
