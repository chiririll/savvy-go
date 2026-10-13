import { apiClient } from './client'
import type { PasswordTokenPreview } from '@/types/users'
import {
    AuthResponse,
    AuthStatus,
    LoginCredentials,
    RegisterData,
    User,
    MeResponse,
    TwoFactorAuthResponse,
    TwoFactorStatus,
    TwoFactorEnableResponse,
    TwoFactorConfirmResponse,
    TwoFactorRecoveryCodesResponse,
    TwoFactorRegenerateResponse,
} from '@/types'

export const authApi = {
    status: async (): Promise<AuthStatus> => {
        const response = await apiClient.get('/auth/status')
        return response.data
    },

    login: async (credentials: LoginCredentials): Promise<AuthResponse | TwoFactorAuthResponse> => {
        const response = await apiClient.post('/auth/login', credentials)
        return response.data
    },

    /** Checks the first-run password without using it up; rejects when it is wrong. */
    verifyBootstrap: async (bootstrapPassword: string): Promise<void> => {
        await apiClient.post('/auth/bootstrap/verify', { bootstrap_password: bootstrapPassword })
    },

    register: async (data: RegisterData): Promise<AuthResponse> => {
        const response = await apiClient.post('/auth/register', data)
        return response.data
    },

    previewPasswordToken: async (token: string): Promise<PasswordTokenPreview> => {
        const response = await apiClient.get(`/auth/password/${token}`)
        return response.data
    },

    acceptPasswordToken: async (
        token: string,
        password: string,
        passwordConfirmation: string,
    ): Promise<AuthResponse | TwoFactorAuthResponse> => {
        const response = await apiClient.post(`/auth/password/${token}`, {
            password,
            password_confirmation: passwordConfirmation,
        })
        return response.data
    },

    me: async (): Promise<MeResponse> => {
        const response = await apiClient.get('/auth/me')
        return response.data
    },

    logout: async (): Promise<void> => {
        await apiClient.post('/auth/logout')
    },

    changePassword: async (data: {
        current_password: string
        password: string
        password_confirmation: string
    }): Promise<{ message: string }> => {
        const response = await apiClient.put('/auth/password', data)
        return response.data
    },

    logoutOthers: async (): Promise<{ message: string; revoked: number }> => {
        const response = await apiClient.post('/auth/logout-others')
        return response.data
    },

    // 2FA Methods
    twoFactorStatus: async (): Promise<TwoFactorStatus> => {
        const response = await apiClient.get('/auth/2fa/status')
        return response.data
    },

    twoFactorEnable: async (): Promise<TwoFactorEnableResponse> => {
        const response = await apiClient.post('/auth/2fa/enable')
        return response.data
    },

    twoFactorConfirm: async (code: string): Promise<TwoFactorConfirmResponse> => {
        const response = await apiClient.post('/auth/2fa/confirm', { code })
        return response.data
    },

    twoFactorDisable: async (code: string): Promise<{ message: string }> => {
        const response = await apiClient.post('/auth/2fa/disable', { code })
        return response.data
    },

    twoFactorVerify: async (twoFactorToken: string, code: string, rememberMe = false): Promise<AuthResponse> => {
        const response = await apiClient.post('/auth/2fa/verify', {
            two_factor_token: twoFactorToken,
            code,
            remember_me: rememberMe,
        })
        return response.data
    },

    twoFactorRecoveryCodes: async (): Promise<TwoFactorRecoveryCodesResponse> => {
        const response = await apiClient.get('/auth/2fa/recovery-codes')
        return response.data
    },

    twoFactorRegenerateRecoveryCodes: async (code: string): Promise<TwoFactorRegenerateResponse> => {
        const response = await apiClient.post('/auth/2fa/recovery-codes/regenerate', { code })
        return response.data
    },
}
