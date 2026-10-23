import type { Meta, StoryObj } from '@storybook/react-vite'
import { Route, Routes, useLocation } from 'react-router-dom'
import { expect, userEvent, waitFor, within } from 'storybook/test'

import { fail, hang, raw, reply } from '@/test/api-mocks'
import { useAuthStore } from '@/stores/auth'

import InvitePage from './invite'

const preview = (overrides = {}) =>
    reply('get', '/invitations/tok', {
        spaceName: 'Family',
        role: 'user',
        inviter: 'Kirill',
        email: 'alex@example.com',
        expiresAt: '2099-01-01T00:00:00Z',
        canRegister: true,
        ...overrides,
    })

const Where = () => <p data-testid="where">{useLocation().pathname}</p>

const meta = {
    title: 'pages/Invite',
    component: InvitePage,
    parameters: { route: '/invite/tok', app: { signedOut: true } },
    render: () => (
        <Routes>
            <Route path="/invite/:token" element={<InvitePage />} />
            <Route path="*" element={<Where />} />
        </Routes>
    ),
    beforeEach: ({ msw }) => {
        msw.use(preview(), raw('get', '/auth/me', { message: 'Unauthenticated' }, 401))
    },
} satisfies Meta<typeof InvitePage>

export default meta
type Story = StoryObj<typeof meta>

export const SignedOut: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByLabelText(/password/i)).toBeVisible()
        // The invited address is fixed.
        await expect(canvas.getByLabelText(/email/i)).toHaveAttribute('readonly')
        await expect(canvas.getByRole('link', { name: /sign in/i })).toHaveAttribute('href', '/login')
    },
}
export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/invitations/tok')) }

export const WithoutRegistration: Story = {
    beforeEach: ({ msw }) => msw.use(preview({ canRegister: false })),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('link', { name: /sign in/i })).toBeVisible()
        await expect(within(canvasElement).queryByLabelText(/password/i)).toBeNull()
    },
}

export const InvalidLink: Story = {
    beforeEach: ({ msw }) => msw.use(fail('get', '/invitations/tok', 'Not found', 404)),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/invalid|expired/i)).toBeVisible()
    },
}

export const SignedInJoins: Story = {
    parameters: { app: {} },
    beforeEach: ({ msw }) => {
        useAuthStore.setState({ isLoading: false })
        msw.use(
            raw('get', '/auth/me', {
                user: { id: 1, name: 'Kirill', email: 'kirill@example.com', role: 'admin', emailVerified: true },
            }),
            reply('post', '/invitations/tok/accept', { id: 2, name: 'Family', emoji: '🏡', color: null, role: 'user' }),
        )
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByRole('button', { name: /join/i }))
        await waitFor(() => expect(canvas.getByTestId('where')).toHaveTextContent(/^\/$/))
    },
}
