import type { UserRole } from './users'

export interface User {
    id: number
    name: string
    email: string
    role?: UserRole
    isSsoOnly?: boolean
}

export interface LoginCredentials {
    email: string
    password: string
    remember_me?: boolean
}

export interface RegisterData {
    name: string
    email: string
    password: string
}

export interface AuthResponse {
    user: User
    expires_at: string
    refresh_at: string | null
}

export interface MeResponse {
    user: User | null
    expires_at: string | null
    refresh_at: string | null
}

export interface TwoFactorAuthResponse {
    requires_2fa: true
    two_factor_token: string
}

export interface AuthStatus {
    needs_registration: boolean
    password_login_enabled: boolean
}

// 2FA Types
export interface TwoFactorStatus {
    enabled: boolean
    pending_confirmation: boolean
    recovery_codes_remaining: number | null
}

export interface TwoFactorEnableResponse {
    secret: string
    qr_code_url: string
}

export interface TwoFactorConfirmResponse {
    message: string
    recovery_codes: string[]
}

export interface TwoFactorRecoveryCodesResponse {
    remaining_count: number
}

export interface TwoFactorRegenerateResponse {
    message: string
    recovery_codes: string[]
}

// Passkey (WebAuthn) Types
import type {
    PublicKeyCredentialCreationOptionsJSON,
    PublicKeyCredentialRequestOptionsJSON,
} from '@simplewebauthn/browser'

export interface WebauthnCredentialSummary {
    id: number
    name: string | null
    aaguid: string | null
    lastUsedAt: string | null
    createdAt: string
}

export interface WebauthnRegisterOptions {
    token: string
    options: PublicKeyCredentialCreationOptionsJSON
}

export interface WebauthnLoginOptions {
    token: string
    options: PublicKeyCredentialRequestOptionsJSON
}
