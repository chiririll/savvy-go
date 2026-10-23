import type { Meta, StoryObj } from '@storybook/react-vite'
import { Route, Routes, useLocation } from 'react-router-dom'
import { expect, within } from 'storybook/test'

import { hang, raw, reply } from '@/test/api-mocks'
import { useAuthStore } from '@/stores/auth'

import LoginPage from './login'

const status = (overrides = {}) =>
    raw('get', '/auth/status', { needs_registration: false, password_login_enabled: true, bootstrap_hint: null, ...overrides })

const github = { slug: 'gh', name: 'GitHub', preset: 'github', protocol: 'oauth2' }

const Where = () => <p data-testid="where">{useLocation().pathname}</p>

const meta = {
    title: 'pages/auth/Login',
    component: LoginPage,
    parameters: { route: '/login' },
    render: () => (
        <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="*" element={<Where />} />
        </Routes>
    ),
    beforeEach: ({ msw }) => {
        useAuthStore.setState({ user: null, isAuthenticated: false, isLoading: false })
        msw.use(status(), reply('get', '/auth/sso/providers', []))
    },
} satisfies Meta<typeof LoginPage>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Welcome back')).toBeVisible()
        await expect(canvas.getByLabelText(/email/i)).toBeVisible()
        await expect(canvas.getByRole('button', { name: /^sign in$/i })).toBeVisible()
    },
}
export const Checking: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/auth/status')) }

export const WithSingleSignOn: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/auth/sso/providers', [github])),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('button', { name: /github/i })).toBeVisible()
        await expect(within(canvasElement).getByLabelText(/email/i)).toBeVisible()
    },
}

export const OnlySingleSignOn: Story = {
    beforeEach: ({ msw }) => msw.use(status({ password_login_enabled: false }), reply('get', '/auth/sso/providers', [github])),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('button', { name: /github/i })).toBeVisible()
        await expect(within(canvasElement).queryByLabelText(/email/i)).toBeNull()
    },
}

export const FirstRunGoesToSetup: Story = {
    beforeEach: ({ msw }) => msw.use(status({ needs_registration: true })),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByTestId('where')).toHaveTextContent('/setup')
    },
}
