// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { EmailVerificationGate } from './EmailVerificationGate'
import { useAuthStore } from '@/stores/auth'

const resend = vi.fn()

vi.mock('react-i18next', async (importActual) => ({
    ...(await importActual<typeof import('react-i18next')>()),
    useTranslation: () => ({ t: (key: string) => key }),
}))
vi.mock('@/api', () => ({
    authApi: { resendVerification: () => resend(), logout: vi.fn(), me: vi.fn(), status: vi.fn() },
    webauthnApi: {},
}))
vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

const user = { id: 1, name: 'Ann', email: 'ann@test.com' }

function setSession(opts: { required: boolean; verified?: boolean }) {
    useAuthStore.setState({
        user: { ...user, emailVerified: opts.verified },
        isAuthenticated: true,
        emailVerificationRequired: opts.required,
    })
}

beforeEach(() => {
    resend.mockReset().mockResolvedValue(undefined)
})
afterEach(cleanup)

describe('EmailVerificationGate', () => {
    it('lets everyone in while verification is not required', () => {
        setSession({ required: false, verified: false })
        render(<EmailVerificationGate>app</EmailVerificationGate>)
        expect(screen.getByText('app')).toBeTruthy()
    })

    it('lets a confirmed user in', () => {
        setSession({ required: true, verified: true })
        render(<EmailVerificationGate>app</EmailVerificationGate>)
        expect(screen.getByText('app')).toBeTruthy()
    })

    it('holds back an unconfirmed user and can send the link again', async () => {
        setSession({ required: true, verified: false })
        render(<EmailVerificationGate>app</EmailVerificationGate>)
        expect(screen.queryByText('app')).toBeNull()

        fireEvent.click(screen.getByRole('button', { name: 'verifyEmail.resend' }))
        await waitFor(() => expect(resend).toHaveBeenCalledOnce())
    })

    it('does not guess: a user without the flag (older server) is let in', () => {
        setSession({ required: true, verified: undefined })
        render(<EmailVerificationGate>app</EmailVerificationGate>)
        expect(screen.getByText('app')).toBeTruthy()
    })
})
