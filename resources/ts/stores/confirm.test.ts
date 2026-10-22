import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { User } from '@/types'

const mocks = vi.hoisted(() => ({
    status: { enabled: false },
    authApi: { twoFactorStatus: vi.fn() },
}))

vi.mock('@/api', () => ({ authApi: mocks.authApi, webauthnApi: {} }))
vi.mock('@/api/client', () => ({ setOnUnauthorized: vi.fn(), setOnEmailUnverified: vi.fn() }))

import { useAuthStore } from './auth'
import {
    cancelConfirmation,
    isConfirmCancelled,
    submitConfirmation,
    useConfirmStore,
    withConfirmation,
} from './confirm'

const user = (extra: Partial<User> = {}) => ({ id: 1, name: 'Ann', email: 'ann@example.com', ...extra }) as User
const pending = () => useConfirmStore.getState().pending

// withConfirmation looks the user up first, so the dialog opens a tick later.
const opened = () => vi.waitFor(() => expect(pending()).not.toBeNull())

beforeEach(() => {
    vi.clearAllMocks()
    mocks.status.enabled = false
    mocks.authApi.twoFactorStatus.mockImplementation(async () => ({ ...mocks.status }))
    useConfirmStore.setState({ pending: null })
    useAuthStore.setState({ user: user() })
})

describe('withConfirmation', () => {
    it('asks for the password and sends what was typed', async () => {
        const run = vi.fn().mockResolvedValue('done')
        const result = withConfirmation(run)
        await opened()
        expect(pending()?.mode).toBe('password')

        await expect(submitConfirmation({ current_password: 'secret1' })).resolves.toBeNull()
        await expect(result).resolves.toBe('done')
        expect(run).toHaveBeenCalledWith({ current_password: 'secret1' })
        expect(pending()).toBeNull()
    })

    it('asks for a code when 2FA is on', async () => {
        mocks.status.enabled = true
        const result = withConfirmation(vi.fn().mockResolvedValue('done'))
        await opened()
        expect(pending()?.mode).toBe('code')
        await submitConfirmation({ code: '123456' })
        await result
    })

    it('does not ask an account that has nothing to confirm with', async () => {
        useAuthStore.setState({ user: user({ isSsoOnly: true }) })
        const run = vi.fn().mockResolvedValue('done')

        await expect(withConfirmation(run)).resolves.toBe('done')
        expect(run).toHaveBeenCalledWith({})
        expect(pending()).toBeNull()
    })

    it('keeps the dialog open after a wrong answer and takes another', async () => {
        const run = vi
            .fn()
            .mockRejectedValueOnce({ reason: 'confirmation_invalid', code: '422', message: 'Wrong.' })
            .mockResolvedValueOnce('done')
        const result = withConfirmation(run)
        await opened()

        await expect(submitConfirmation({ current_password: 'nope' })).resolves.toBe('Wrong.')
        expect(pending()).not.toBeNull()

        await expect(submitConfirmation({ current_password: 'secret1' })).resolves.toBeNull()
        await expect(result).resolves.toBe('done')
    })

    it('hands any other failure to the caller and closes', async () => {
        const failure = { code: '422', message: 'You can create at most 50 API tokens.' }
        const result = withConfirmation(vi.fn().mockRejectedValue(failure))
        const settled = expect(result).rejects.toBe(failure)
        await opened()

        await submitConfirmation({ current_password: 'secret1' })
        await settled
        expect(pending()).toBeNull()
    })

    it('rejects as cancelled when the dialog is closed', async () => {
        const run = vi.fn()
        const result = withConfirmation(run)
        const settled = expect(result).rejects.toSatisfy(isConfirmCancelled)
        await opened()

        cancelConfirmation()
        await settled
        expect(run).not.toHaveBeenCalled()
    })
})
