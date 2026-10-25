import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { hang, reply } from '@/test/api-mocks'
import { provider } from '@/components/features/sso/fixtures'

import AdminSsoPage from './sso'

const meta = {
    title: 'pages/admin/Sso',
    component: AdminSsoPage,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) =>
        msw.use(reply('get', '/identity-providers', [provider(), provider({ id: 2, name: 'Off', slug: 'off', enabled: false })])),
} satisfies Meta<typeof AdminSsoPage>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Company GitHub')).toBeVisible()
        // With providers in place the page offers to add another.
        await expect(canvas.getByRole('link', { name: /add/i })).toHaveAttribute('href', '/admin/providers/create')
    },
}
export const Empty: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/identity-providers', [])),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('heading', { level: 2 })).toBeVisible()
    },
}
export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/identity-providers')) }
