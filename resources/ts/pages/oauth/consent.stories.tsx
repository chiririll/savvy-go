import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { capabilities } from '@/components/features/api-tokens/story-fixtures'
import { fail, hang, reply } from '@/test/api-mocks'

import OAuthConsentPage from './consent'

const query =
    'client_id=c1&redirect_uri=https%3A%2F%2Fclaude.ai%2Fcallback&state=s&code_challenge=abc&code_challenge_method=S256'

const info = {
    client_name: 'Claude',
    resources: ['transactions'],
    spaces: [
        { id: 1, name: 'Home', role: 'admin' },
        { id: 2, name: 'Trip', role: 'user' },
    ],
}

const meta = {
    title: 'pages/oauth/Consent',
    component: OAuthConsentPage,
    parameters: { a11y: overlayA11y, route: `/oauth/consent?${query}` },
    beforeEach: ({ msw }) =>
        msw.use(reply('get', '/oauth/authorize', info), reply('get', '/auth/api-token-capabilities', capabilities)),
} satisfies Meta<typeof OAuthConsentPage>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByRole('button', { name: /allow/i })).toBeVisible()
        await expect(canvas.getByRole('button', { name: /cancel/i })).toBeVisible()
        // The user can see where the browser goes back to.
        await expect(canvas.getByText(/claude\.ai/)).toBeVisible()
    },
}
export const InRussian: Story = { globals: { locale: 'ru' } }
export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/oauth/authorize')) }

export const UnknownClient: Story = {
    beforeEach: ({ msw }) => msw.use(fail('get', '/oauth/authorize', 'Unknown client', 404)),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/invalid|unknown|not valid/i)).toBeVisible()
        await expect(within(canvasElement).queryByRole('button', { name: /allow/i })).toBeNull()
    },
}

export const MissingParameters: Story = {
    parameters: { route: '/oauth/consent' },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/cannot be used/i)).toBeVisible()
    },
}
