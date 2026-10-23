import type { Meta, StoryObj } from '@storybook/react-vite'
import { Route, Routes, useLocation } from 'react-router-dom'
import { expect, userEvent, waitFor, within } from 'storybook/test'

import { fail, hang, raw } from '@/test/api-mocks'

import AdminSetupPage from './admin'

const status = (overrides = {}) =>
    raw('get', '/auth/status', {
        needs_registration: true,
        password_login_enabled: true,
        bootstrap_hint: { environment: 'docker', file: '/data/bootstrap-password' },
        ...overrides,
    })

const Where = () => <p data-testid="where">{useLocation().pathname}</p>

const meta = {
    title: 'pages/onboarding/AdminSetup',
    component: AdminSetupPage,
    parameters: { route: '/setup', app: { signedOut: true } },
    render: () => (
        <Routes>
            <Route path="/setup" element={<AdminSetupPage />} />
            <Route path="*" element={<Where />} />
        </Routes>
    ),
    beforeEach: ({ msw }) => msw.use(status(), raw('post', '/auth/bootstrap/verify', { message: 'ok' })),
} satisfies Meta<typeof AdminSetupPage>

export default meta
type Story = StoryObj<typeof meta>

export const Bootstrap: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByLabelText(/bootstrap password/i)).toBeVisible()
        await expect(canvas.getByRole('button', { name: /create admin/i })).toBeVisible()
    },
}
export const InRussian: Story = { globals: { locale: 'ru' } }
export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/auth/status')) }

export const AlreadySetUp: Story = {
    beforeEach: ({ msw }) => msw.use(status({ needs_registration: false, bootstrap_hint: null })),
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(within(canvasElement).getByTestId('where')).toHaveTextContent('/login'))
    },
}

export const WrongBootstrapPassword: Story = {
    beforeEach: ({ msw }) => msw.use(fail('post', '/auth/bootstrap/verify', 'Wrong password', 422)),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.type(await canvas.findByLabelText(/bootstrap password/i), 'ABCDE')
        await userEvent.click(canvas.getByRole('button', { name: /create admin/i }))
        await expect(await canvas.findByText(/wrong password|invalid|incorrect/i)).toBeVisible()
    },
}

export const AdminFields: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.type(await canvas.findByLabelText(/bootstrap password/i), 'ABCDE')
        await userEvent.click(canvas.getByRole('button', { name: /create admin/i }))
        await expect(await canvas.findByLabelText(/your name/i)).toBeVisible()
        await expect(canvas.getByLabelText(/^email/i)).toBeVisible()
    },
}
