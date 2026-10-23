import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { useLocation } from 'react-router-dom'

import { raw } from '@/test/api-mocks'
import { useAuthStore } from '@/stores/auth'

import Setup2FAPage from './setup-2fa'

const secret = { secret: 'JBSWY3DPEHPK3PXP', qr_code_url: 'otpauth://totp/GoSavvy:kirill?secret=JBSWY3DPEHPK3PXP' }
const codes = ['aaaa-1111', 'bbbb-2222', 'cccc-3333', 'dddd-4444']

// Shows where the page sent the user.
const Place = () => <p data-testid="place">{useLocation().pathname}</p>

const meta = {
    title: 'pages/auth/Setup2FA',
    component: Setup2FAPage,
    parameters: { route: '/setup-2fa' },
    render: () => (
        <>
            <Setup2FAPage />
            <Place />
        </>
    ),
    beforeEach: ({ msw }) => {
        // Only a fresh registration may see the page.
        sessionStorage.setItem('just_registered', 'true')
        // A single-sign-on account has nothing to confirm with, so enabling goes straight through.
        const user = useAuthStore.getState().user
        if (user) useAuthStore.setState({ user: { ...user, isSsoOnly: true } })
        msw.use(
            raw('get', '/auth/2fa/status', { enabled: false, pending_confirmation: false, recovery_codes_remaining: null }),
            raw('post', '/auth/2fa/enable', secret),
            raw('post', '/auth/2fa/confirm', { message: 'ok', recovery_codes: codes }),
        )
    },
} satisfies Meta<typeof Setup2FAPage>

export default meta
type Story = StoryObj<typeof meta>

export const Intro: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByRole('button', { name: /enable/i })).toBeVisible()
        await expect(canvas.getByRole('button', { name: /skip/i })).toBeVisible()
    },
}

export const SkipLeavesForTheDashboard: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /skip/i }))
        await waitFor(() => expect(within(canvasElement).getByTestId('place')).toHaveTextContent(/^\/$/))
        await expect(sessionStorage.getItem('just_registered')).toBeNull()
    },
}

export const WalksThroughTheSteps: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByRole('button', { name: /enable/i }))
        // Step one: the secret to scan or type in.
        await expect(await canvas.findByText(secret.secret)).toBeVisible()
        // Back to the start and forward again.
        await userEvent.click(canvas.getByRole('button', { name: /back/i }))
        await userEvent.click(await canvas.findByRole('button', { name: /enable/i }))
        await userEvent.click(await canvas.findByRole('button', { name: /continue/i }))
        // Step two: the code from the authenticator.
        await expect(canvas.getByRole('button', { name: /^verify/i })).toBeDisabled()
        await userEvent.click(await canvas.findByRole('textbox'))
        await userEvent.keyboard('123456')
        await userEvent.click(canvas.getByRole('button', { name: /^verify/i }))
        // Step three: the recovery codes.
        for (const code of codes) await expect(await canvas.findByText(code)).toBeVisible()
        await userEvent.click(canvas.getByRole('button', { name: /done/i }))
        await waitFor(() => expect(canvas.getByTestId('place')).toHaveTextContent(/^\/$/))
    },
}

export const NotJustRegistered: Story = {
    beforeEach: () => sessionStorage.removeItem('just_registered'),
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(within(canvasElement).getByTestId('place')).toHaveTextContent(/^\/$/))
    },
}
