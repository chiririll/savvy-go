import type { Meta, StoryObj } from '@storybook/react-vite'
import { Route, Routes, useLocation } from 'react-router-dom'
import { expect, userEvent, waitFor, within } from 'storybook/test'

import { fail, hang, raw } from '@/test/api-mocks'

import VerifyEmailPage from './verify-email'

const Where = () => <p data-testid="where">{useLocation().pathname}</p>

const meta = {
    title: 'pages/auth/VerifyEmail',
    component: VerifyEmailPage,
    parameters: { route: '/verify-email/abc' },
    render: () => (
        <Routes>
            <Route path="/verify-email/:token" element={<VerifyEmailPage />} />
            <Route path="*" element={<Where />} />
        </Routes>
    ),
    beforeEach: ({ msw }) =>
        msw.use(raw('post', '/auth/verify-email', { message: 'ok' }), fail('get', '/auth/me', 'Unauthenticated', 401)),
} satisfies Meta<typeof VerifyEmailPage>

export default meta
type Story = StoryObj<typeof meta>

export const Confirmed: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByRole('button', { name: /continue/i }))
        await waitFor(() => expect(canvas.getByTestId('where')).toHaveTextContent(/^\/$/))
    },
}
export const Working: Story = { beforeEach: ({ msw }) => msw.use(hang('post', '/auth/verify-email')) }

export const InvalidLink: Story = {
    beforeEach: ({ msw }) => msw.use(fail('post', '/auth/verify-email', 'Invalid link', 422)),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('button', { name: /continue/i })).toBeVisible()
    },
}
