import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { User } from '@/types'

const mocks = vi.hoisted(() => ({
    authApi: {
        login: vi.fn(),
        twoFactorVerify: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
        me: vi.fn(),
    },
    webauthnApi: { loginOptions: vi.fn(), loginVerify: vi.fn() },
    startAuthentication: vi.fn(),
    sessionStorage: { setItem: vi.fn() },
    unauthorized: { handler: null as null | (() => void) },
    unverified: { handler: null as null | (() => void) },
}))

vi.mock('@/api', () => ({ authApi: mocks.authApi, webauthnApi: mocks.webauthnApi }))
vi.mock('@/api/client', () => ({
    setOnUnauthorized: (fn: () => void) => {
        mocks.unauthorized.handler = fn
    },
    setOnEmailUnverified: (fn: () => void) => {
        mocks.unverified.handler = fn
    },
}))
vi.mock('@simplewebauthn/browser', () => ({ startAuthentication: mocks.startAuthentication }))
vi.stubGlobal('sessionStorage', mocks.sessionStorage)

import { useAuthStore } from './auth'
import { queryClient } from '@/lib/query-client'
import { useSpaceStore } from './space'

const user = { id: 1, name: 'Ann', email: 'ann@example.com', role: 'user' } as User
const session = { user, expires_at: '2026-01-02T00:00:00Z', refresh_at: '2026-01-01T12:00:00Z' }
const state = () => useAuthStore.getState()

beforeEach(() => {
    vi.clearAllMocks()
    useAuthStore.setState(useAuthStore.getInitialState(), true)
})

describe('initial state', () => {
    it('starts loading and signed out', () => {
        expect(state()).toMatchObject({ user: null, isLoading: true, isAuthenticated: false, sessionExpired: false })
    })
})

describe('login', () => {
    it('applies the session, including its expiry times', async () => {
        mocks.authApi.login.mockResolvedValue(session)

        await expect(state().login({ email: 'a', password: 'b' })).resolves.toEqual({ success: true })

        expect(state()).toMatchObject({
            user,
            isAuthenticated: true,
            isLoading: false,
            expiresAt: '2026-01-02T00:00:00Z',
            refreshAt: '2026-01-01T12:00:00Z',
        })
    })

    it('stops at the second factor without signing in', async () => {
        mocks.authApi.login.mockResolvedValue({ requires_2fa: true, two_factor_token: 'tok' })

        await expect(state().login({ email: 'a', password: 'b' })).resolves.toEqual({
            success: false,
            requires_2fa: true,
            two_factor_token: 'tok',
        })
        expect(state().isAuthenticated).toBe(false)
        expect(state().user).toBeNull()
    })

    it('lets a failed login reach the caller and leaves the state alone', async () => {
        mocks.authApi.login.mockRejectedValue(new Error('bad credentials'))

        await expect(state().login({ email: 'a', password: 'b' })).rejects.toThrow('bad credentials')
        expect(state().isAuthenticated).toBe(false)
    })

    it('treats a missing expiry as null', async () => {
        mocks.authApi.login.mockResolvedValue({ user })
        await state().login({ email: 'a', password: 'b' })
        expect(state()).toMatchObject({ expiresAt: null, refreshAt: null })
    })
})

describe('loginWith2FA', () => {
    it('verifies the code and signs in', async () => {
        mocks.authApi.twoFactorVerify.mockResolvedValue(session)

        await state().loginWith2FA('tok', '123456')

        expect(mocks.authApi.twoFactorVerify).toHaveBeenCalledWith('tok', '123456', false)
        expect(state().isAuthenticated).toBe(true)
    })

    it('passes remember-me through', async () => {
        mocks.authApi.twoFactorVerify.mockResolvedValue(session)
        await state().loginWith2FA('tok', '123456', true)
        expect(mocks.authApi.twoFactorVerify).toHaveBeenCalledWith('tok', '123456', true)
    })
})

describe('loginWithPasskey', () => {
    it('runs the WebAuthn ceremony and signs in', async () => {
        mocks.webauthnApi.loginOptions.mockResolvedValue({ token: 'ch', options: { challenge: 'x' } })
        mocks.startAuthentication.mockResolvedValue({ id: 'cred' })
        mocks.webauthnApi.loginVerify.mockResolvedValue(session)

        await state().loginWithPasskey({ twoFactorToken: 'tf', useAutofill: true })

        expect(mocks.webauthnApi.loginOptions).toHaveBeenCalledWith('tf')
        expect(mocks.startAuthentication).toHaveBeenCalledWith({
            optionsJSON: { challenge: 'x' },
            useBrowserAutofill: true,
        })
        expect(mocks.webauthnApi.loginVerify).toHaveBeenCalledWith('ch', { id: 'cred' }, 'tf')
        expect(state().isAuthenticated).toBe(true)
    })

    it('does not use browser autofill unless asked', async () => {
        mocks.webauthnApi.loginOptions.mockResolvedValue({ token: 'ch', options: {} })
        mocks.startAuthentication.mockResolvedValue({})
        mocks.webauthnApi.loginVerify.mockResolvedValue(session)

        await state().loginWithPasskey()

        expect(mocks.startAuthentication).toHaveBeenCalledWith({ optionsJSON: {}, useBrowserAutofill: false })
        expect(mocks.webauthnApi.loginVerify).toHaveBeenCalledWith('ch', {}, undefined)
    })

    it('stays signed out when the ceremony is cancelled', async () => {
        mocks.webauthnApi.loginOptions.mockResolvedValue({ token: 'ch', options: {} })
        mocks.startAuthentication.mockRejectedValue(new Error('cancelled'))

        await expect(state().loginWithPasskey()).rejects.toThrow('cancelled')
        expect(state().isAuthenticated).toBe(false)
    })
})

describe('register', () => {
    it('signs in and remembers that the account is new', async () => {
        mocks.authApi.register.mockResolvedValue(session)

        await state().register({ name: 'Ann', email: 'a', password: 'b' } as never)

        expect(mocks.sessionStorage.setItem).toHaveBeenCalledWith('just_registered', 'true')
        expect(state().isAuthenticated).toBe(true)
    })
})

describe('signing in as another account without logging out', () => {
    const other = { ...session, user: { ...user, id: 2 } }

    beforeEach(() => {
        queryClient.clear()
        useSpaceStore.setState({ currentId: null, ownerId: null })
    })

    it('drops the previous account data when the session expired and someone else logs in', async () => {
        state().applySession(session as never)
        queryClient.setQueryData(['dashboard'], { stale: true })
        useSpaceStore.getState().setCurrentId(7)
        state().expire()

        mocks.authApi.login.mockResolvedValue(other)
        await state().login({ email: 'b', password: 'b' })

        expect(queryClient.getQueryData(['dashboard'])).toBeUndefined()
        expect(useSpaceStore.getState().currentId).toBeNull()
    })

    it('drops a space remembered by another account after a reload', async () => {
        useSpaceStore.setState({ currentId: 7, ownerId: 1 })
        mocks.authApi.me.mockResolvedValue({ user: other.user })

        await state().checkAuth()

        expect(useSpaceStore.getState().currentId).toBeNull()
        expect(useSpaceStore.getState().ownerId).toBe(2)
    })

    it('keeps the space when the same account signs in again', async () => {
        useSpaceStore.setState({ currentId: 7, ownerId: 1 })
        mocks.authApi.login.mockResolvedValue(session)

        await state().login({ email: 'a', password: 'b' })

        expect(useSpaceStore.getState().currentId).toBe(7)
    })
})

describe('logout', () => {
    beforeEach(() => state().applySession(session as never))

    it('clears the session', async () => {
        mocks.authApi.logout.mockResolvedValue(undefined)
        await state().logout()
        expect(state()).toMatchObject({ user: null, isAuthenticated: false, expiresAt: null, refreshAt: null })
    })

    it('drops cached data and the remembered space so the next login starts clean', async () => {
        mocks.authApi.logout.mockResolvedValue(undefined)
        queryClient.setQueryData(['dashboard'], { stale: true })
        useSpaceStore.getState().setCurrentId(7)
        await state().logout()
        expect(queryClient.getQueryData(['dashboard'])).toBeUndefined()
        expect(useSpaceStore.getState().currentId).toBeNull()
    })

    it('clears the session even if the server call fails', async () => {
        mocks.authApi.logout.mockRejectedValue(new Error('network'))
        await expect(state().logout()).resolves.toBeUndefined()
        expect(state().isAuthenticated).toBe(false)
    })
})

describe('checkAuth', () => {
    it('adopts the user the server reports', async () => {
        mocks.authApi.me.mockResolvedValue(session)
        await state().checkAuth()
        expect(state()).toMatchObject({ user, isAuthenticated: true, isLoading: false, sessionExpired: false })
    })

    it('settles as signed out for a visitor', async () => {
        mocks.authApi.me.mockResolvedValue({ user: null })
        await state().checkAuth()
        expect(state()).toMatchObject({ user: null, isAuthenticated: false, isLoading: false, sessionExpired: false })
    })

    it('settles as signed out if the request fails and nobody was in', async () => {
        mocks.authApi.me.mockRejectedValue(new Error('offline'))
        await state().checkAuth()
        expect(state()).toMatchObject({ isAuthenticated: false, isLoading: false, sessionExpired: false })
    })

    it('flags the session as expired when a signed-in user is unrecognised', async () => {
        state().applySession(session as never)
        mocks.authApi.me.mockResolvedValue({ user: null })

        await state().checkAuth()

        expect(state()).toMatchObject({ sessionExpired: true, isLoading: false, isAuthenticated: true })
    })

    it('flags the session as expired when the signed-in user check fails', async () => {
        state().applySession(session as never)
        mocks.authApi.me.mockRejectedValue(new Error('401'))

        await state().checkAuth()

        expect(state().sessionExpired).toBe(true)
        expect(state().user).toEqual(user)
    })
})

describe('session bookkeeping', () => {
    it('setUser signs the user in and clears the expired flag', () => {
        state().expire()
        state().setUser(user)
        expect(state()).toMatchObject({ user, isAuthenticated: true, sessionExpired: false })
    })

    it('expire marks the session expired and stops loading', () => {
        state().expire()
        expect(state()).toMatchObject({ sessionExpired: true, isLoading: false })
    })

    it('clear resets everything', () => {
        state().applySession(session as never)
        state().expire()
        state().clear()
        expect(state()).toMatchObject({
            user: null,
            isAuthenticated: false,
            sessionExpired: false,
            expiresAt: null,
            refreshAt: null,
        })
    })
})

describe('401 handler', () => {
    it('is registered with the API client', () => {
        expect(mocks.unauthorized.handler).toBeTypeOf('function')
    })

    it('expires a signed-in session', () => {
        state().applySession(session as never)
        mocks.unauthorized.handler!()
        expect(state().sessionExpired).toBe(true)
    })

    it('ignores a 401 for a visitor', () => {
        mocks.unauthorized.handler!()
        expect(state().sessionExpired).toBe(false)
    })
})

describe('email verification', () => {
    it('takes the requirement from the session and drops it on sign-out', () => {
        state().applySession({ ...session, user: { ...user, emailVerified: false }, email_verification_required: true })
        expect(state().emailVerificationRequired).toBe(true)
        state().clear()
        expect(state().emailVerificationRequired).toBe(false)
    })

    it('closes the app when the server refuses an unconfirmed user', () => {
        state().applySession({ ...session, user: { ...user, emailVerified: true } })
        expect(state().emailVerificationRequired).toBe(false)
        mocks.unverified.handler?.()
        expect(state().emailVerificationRequired).toBe(true)
        expect(state().user?.emailVerified).toBe(false)
    })
})
