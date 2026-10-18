import { create } from 'zustand'
import { startAuthentication } from '@simplewebauthn/browser'
import { authApi, webauthnApi } from '@/api'
import { setOnUnauthorized, setOnEmailUnverified } from '@/api/client'
import { User, LoginCredentials, RegisterData, AuthResponse, TwoFactorAuthResponse, MeResponse } from '@/types'

export type LoginResult = { success: true } | { success: false; requires_2fa: true; two_factor_token: string }

interface AuthState {
    user: User | null
    isLoading: boolean
    isAuthenticated: boolean
    sessionExpired: boolean
    expiresAt: string | null
    refreshAt: string | null
    /** The instance requires a confirmed address; with `user.emailVerified` false, the app is closed to them. */
    emailVerificationRequired: boolean

    login: (credentials: LoginCredentials) => Promise<LoginResult>
    loginWith2FA: (twoFactorToken: string, code: string, rememberMe?: boolean) => Promise<void>
    loginWithPasskey: (options?: { twoFactorToken?: string; useAutofill?: boolean }) => Promise<void>
    register: (data: RegisterData) => Promise<void>
    logout: () => Promise<void>
    checkAuth: () => Promise<void>
    applySession: (response: AuthResponse) => void
    setUser: (user: User) => void
    /** The server refused a request because the address is unconfirmed. */
    holdBackForVerification: () => void
    expire: () => void
    clear: () => void
}

function isTwoFactorResponse(response: AuthResponse | TwoFactorAuthResponse): response is TwoFactorAuthResponse {
    return 'requires_2fa' in response && response.requires_2fa === true
}

function sessionTimes(
    response: Pick<MeResponse, 'expires_at' | 'refresh_at'>,
): Pick<AuthState, 'expiresAt' | 'refreshAt'> {
    return {
        expiresAt: response.expires_at ?? null,
        refreshAt: response.refresh_at ?? null,
    }
}

export const useAuthStore = create<AuthState>((set, get) => ({
    user: null,
    isLoading: true,
    isAuthenticated: false,
    sessionExpired: false,
    expiresAt: null,
    refreshAt: null,
    emailVerificationRequired: false,

    login: async (credentials) => {
        const response = await authApi.login(credentials)

        if (isTwoFactorResponse(response)) {
            return { success: false, requires_2fa: true, two_factor_token: response.two_factor_token }
        }

        get().applySession(response)
        return { success: true }
    },

    loginWith2FA: async (twoFactorToken, code, rememberMe = false) => {
        const response = await authApi.twoFactorVerify(twoFactorToken, code, rememberMe)
        get().applySession(response)
    },

    loginWithPasskey: async ({ twoFactorToken, useAutofill } = {}) => {
        const { token, options } = await webauthnApi.loginOptions(twoFactorToken)
        const assertion = await startAuthentication({
            optionsJSON: options,
            useBrowserAutofill: useAutofill ?? false,
        })
        const response = await webauthnApi.loginVerify(token, assertion, twoFactorToken)
        get().applySession(response)
    },

    register: async (data) => {
        const response = await authApi.register(data)
        sessionStorage.setItem('just_registered', 'true')
        get().applySession(response)
    },

    logout: async () => {
        try {
            await authApi.logout()
        } catch {
            // session may already be gone; clear locally regardless
        }
        get().clear()
    },

    checkAuth: async () => {
        try {
            const response = await authApi.me()
            if (response.user) {
                set({
                    user: response.user,
                    isAuthenticated: true,
                    isLoading: false,
                    sessionExpired: false,
                    emailVerificationRequired: response.email_verification_required ?? false,
                    ...sessionTimes(response),
                })
                return
            }
            if (get().isAuthenticated) {
                get().expire()
                set({ isLoading: false })
                return
            }
            set({
                user: null,
                isAuthenticated: false,
                isLoading: false,
                sessionExpired: false,
                expiresAt: null,
                refreshAt: null,
            })
        } catch {
            if (get().isAuthenticated) {
                get().expire()
                set({ isLoading: false })
                return
            }
            set({
                user: null,
                isAuthenticated: false,
                isLoading: false,
                sessionExpired: false,
                expiresAt: null,
                refreshAt: null,
            })
        }
    },

    applySession: (response) =>
        set({
            user: response.user,
            isAuthenticated: true,
            isLoading: false,
            sessionExpired: false,
            emailVerificationRequired: response.email_verification_required ?? false,
            ...sessionTimes(response),
        }),

    holdBackForVerification: () =>
        set((state) => ({
            emailVerificationRequired: true,
            user: state.user ? { ...state.user, emailVerified: false } : state.user,
        })),

    setUser: (user) => set({ user, isAuthenticated: true, isLoading: false, sessionExpired: false }),

    expire: () => {
        if (get().sessionExpired) {
            return
        }
        set({ sessionExpired: true, isLoading: false })
    },

    clear: () =>
        set({
            user: null,
            isAuthenticated: false,
            isLoading: false,
            sessionExpired: false,
            expiresAt: null,
            refreshAt: null,
            emailVerificationRequired: false,
        }),
}))

setOnUnauthorized(() => {
    const state = useAuthStore.getState()
    if (state.isAuthenticated) {
        state.expire()
    }
})

setOnEmailUnverified(() => useAuthStore.getState().holdBackForVerification())

export const useUser = () => useAuthStore((state) => state.user)
export const useIsAuthenticated = () => useAuthStore((state) => state.isAuthenticated)
export const useAuthLoading = () => useAuthStore((state) => state.isLoading)
export const useSessionExpired = () => useAuthStore((state) => state.sessionExpired)
