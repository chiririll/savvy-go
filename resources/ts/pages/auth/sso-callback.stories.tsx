import type { Meta, StoryObj } from '@storybook/react-vite'
import { Route, Routes, useLocation } from 'react-router-dom'
import { expect, waitFor, within } from 'storybook/test'

import { fail, hang, raw } from '@/test/api-mocks'
import { useAuthStore } from '@/stores/auth'

import SsoCallbackPage from './sso-callback'

const session = {
    user: { id: 1, name: 'Kirill', email: 'kirill@example.com', role: 'admin', emailVerified: true },
    expires_at: '2099-01-01T00:00:00Z',
    refresh_at: null,
}

const Where = () => {
    const location = useLocation()
    return <p data-testid="where">{location.pathname + (location.state ? ' with state' : '')}</p>
}

const meta = {
    tags: ['desktop-only'],
    title: 'pages/auth/SsoCallback',
    component: SsoCallbackPage,
    parameters: { route: '/sso/callback?ticket=t1' },
    render: () => (
        <Routes>
            <Route path="/sso/callback" element={<SsoCallbackPage />} />
            <Route path="*" element={<Where />} />
        </Routes>
    ),
    beforeEach: ({ msw }) => {
        useAuthStore.setState({ user: null, isAuthenticated: false, isLoading: false })
        msw.use(raw('post', '/auth/sso/exchange', session))
    },
} satisfies Meta<typeof SsoCallbackPage>

export default meta
type Story = StoryObj<typeof meta>

const expectWhere = (canvasElement: HTMLElement, text: string | RegExp) =>
    waitFor(() => expect(within(canvasElement).getByTestId('where')).toHaveTextContent(text))

export const SignsIn: Story = {
    play: async ({ canvasElement }) => {
        await expectWhere(canvasElement, /^\/$/)
        await expect(useAuthStore.getState().isAuthenticated).toBe(true)
    },
}

export const Exchanging: Story = { beforeEach: ({ msw }) => msw.use(hang('post', '/auth/sso/exchange')) }

export const AsksForTheSecondFactor: Story = {
    beforeEach: ({ msw }) => msw.use(raw('post', '/auth/sso/exchange', { requires_2fa: true, two_factor_token: 'tok' })),
    play: async ({ canvasElement }) => expectWhere(canvasElement, '/login with state'),
}

export const ExchangeRefused: Story = {
    beforeEach: ({ msw }) => msw.use(fail('post', '/auth/sso/exchange', 'Ticket expired', 400)),
    play: async ({ canvasElement }) => {
        await expectWhere(canvasElement, '/login')
        await expect(useAuthStore.getState().isAuthenticated).toBe(false)
    },
}

export const ProviderError: Story = {
    parameters: { route: '/sso/callback?error=access_denied' },
    play: async ({ canvasElement }) => expectWhere(canvasElement, '/login'),
}

export const NoTicket: Story = {
    parameters: { route: '/sso/callback' },
    play: async ({ canvasElement }) => expectWhere(canvasElement, '/login'),
}
