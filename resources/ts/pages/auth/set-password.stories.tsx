import type { Meta, StoryObj } from '@storybook/react-vite'
import { Route, Routes, useLocation } from 'react-router-dom'
import { expect, userEvent, waitFor, within } from 'storybook/test'

import { fail, hang, raw } from '@/test/api-mocks'
import { useAuthStore } from '@/stores/auth'

import SetPasswordPage from './set-password'

const session = {
    user: { id: 2, name: 'Alex', email: 'alex@example.com', role: 'user', emailVerified: true },
    expires_at: '2099-01-01T00:00:00Z',
    refresh_at: null,
}

const preview = (isInactive: boolean) =>
    raw('get', '/auth/password/tok', { name: 'Alex', email: 'alex@example.com', isInactive, expiresAt: '2099-01-01T00:00:00Z' })

const Where = () => <p data-testid="where">{useLocation().pathname}</p>

const meta = {
    title: 'pages/auth/SetPassword',
    component: SetPasswordPage,
    parameters: { route: '/set-password/tok' },
    render: () => (
        <Routes>
            <Route path="/set-password/:token" element={<SetPasswordPage />} />
            <Route path="*" element={<Where />} />
        </Routes>
    ),
    beforeEach: ({ msw }) => {
        useAuthStore.setState({ user: null, isAuthenticated: false, isLoading: false })
        msw.use(preview(false))
    },
} satisfies Meta<typeof SetPasswordPage>

export default meta
type Story = StoryObj<typeof meta>

export const Reset: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Set a new password')).toBeVisible()
        await expect(canvas.getByText('Set a password for alex@example.com')).toBeVisible()
    },
}
export const InRussian: Story = { globals: { locale: 'ru' } }

export const Invite: Story = {
    beforeEach: ({ msw }) => msw.use(preview(true)),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Choose your password')).toBeVisible()
    },
}

export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/auth/password/tok')) }

export const InvalidLink: Story = {
    beforeEach: ({ msw }) => msw.use(fail('get', '/auth/password/tok', 'Expired', 404)),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Link is invalid')).toBeVisible()
        await userEvent.click(canvas.getByRole('button', { name: /back to sign in/i }))
        await waitFor(() => expect(canvas.getByTestId('where')).toHaveTextContent('/login'))
    },
}

export const SavesAndSignsIn: Story = {
    beforeEach: ({ msw }) => msw.use(raw('post', '/auth/password/tok', session)),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.type(await canvas.findByLabelText(/^password/i), 'correct horse')
        await userEvent.type(canvas.getByLabelText(/confirm/i), 'correct horse')
        await userEvent.click(canvas.getByRole('button', { name: /save password/i }))
        await waitFor(() => expect(canvas.getByTestId('where')).toHaveTextContent(/^\/$/))
        await expect(useAuthStore.getState().isAuthenticated).toBe(true)
    },
}
