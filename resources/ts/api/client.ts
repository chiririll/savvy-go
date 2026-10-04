import axios, { AxiosInstance, AxiosRequestConfig, InternalAxiosRequestConfig } from 'axios'
import { toast } from 'sonner'
import i18n from '@/lib/i18n'
import { getApiErrorMessage } from '@/lib/api-error'
import { ApiError } from '@/types'
import { spacePath } from '@/lib/space-path'
import { currentSpaceId } from '@/stores/space'

interface WrappedResponse<T> {
    data: T
    message?: string
    meta?: {
        total?: number
        page?: number
        limit?: number
    }
}

interface RetryableRequestConfig extends InternalAxiosRequestConfig {
    _csrfRetry?: boolean
}

// Soft, router-driven sign-out: the auth store registers this so a 401 never
// hard-reloads the page. The frontend holds no token and never refreshes.
let onUnauthorized: (() => void) | null = null

export const setOnUnauthorized = (handler: () => void) => {
    onUnauthorized = handler
}

const AUTH_EXEMPT_401 = [
    '/auth/login',
    '/auth/register',
    '/auth/2fa/verify',
    '/auth/webauthn/login',
    '/auth/sso/',
    '/auth/logout',
]

function requestUrl(config?: { url?: string; baseURL?: string }): string {
    return `${config?.url ?? ''}`
}

function isAuthExempt(url: string): boolean {
    return AUTH_EXEMPT_401.some((path) => url.includes(path))
}

const createApiClient = (baseURL: string): AxiosInstance => {
    const client = axios.create({
        baseURL,
        timeout: 10000,
        withCredentials: true,
        xsrfCookieName: 'svy_csrf',
        xsrfHeaderName: 'X-CSRF-Token',
        headers: {
            'Content-Type': 'application/json',
        },
    })

    client.interceptors.request.use((config) => {
        config.headers['X-Locale'] = i18n.resolvedLanguage ?? i18n.language
        // Data of a space lives under /spaces/{id}; SpaceGate renders pages
        // only once the current space is known.
        if (config.url) config.url = spacePath(config.url, currentSpaceId())
        return config
    })

    client.interceptors.response.use(
        (response) => response,
        (error) => {
            const url = requestUrl(error.config)
            const status = error.response?.status

            if (status === 401) {
                if (!isAuthExempt(url)) {
                    onUnauthorized?.()
                }
                return Promise.reject(error)
            }

            if (status === 419) {
                const config = error.config as RetryableRequestConfig | undefined
                if (config && !config._csrfRetry && !isAuthExempt(url)) {
                    config._csrfRetry = true
                    return client.request(config)
                }
            }

            const apiError: ApiError = {
                message: error.response?.data?.message || i18n.t('errors.generic'),
                code: status?.toString() || 'UNKNOWN',
                details: error.response?.data?.errors ?? error.response?.data?.details,
            }
            apiError.message = getApiErrorMessage(apiError, i18n.t('errors.generic'))

            // Show toast for non-validation errors
            if (status !== 422) {
                toast.error(apiError.message)
            }

            return Promise.reject(apiError)
        }
    )

    return client
}

export const apiClient = createApiClient(
    import.meta.env.VITE_API_URL || '/api'
)

// Helper to unwrap response
const unwrap = <T>(response: WrappedResponse<T> | T): T => {
    if (response && typeof response === 'object' && 'data' in response) {
        return (response as WrappedResponse<T>).data
    }
    return response as T
}

export const api = {
    get: <T>(url: string, config?: AxiosRequestConfig) =>
        apiClient.get<WrappedResponse<T> | T>(url, config)
            .then((res) => unwrap<T>(res.data)),

    post: <T, D = unknown>(url: string, data?: D, config?: AxiosRequestConfig) =>
        apiClient.post<WrappedResponse<T> | T>(url, data, config)
            .then((res) => unwrap<T>(res.data)),

    put: <T, D = unknown>(url: string, data?: D, config?: AxiosRequestConfig) =>
        apiClient.put<WrappedResponse<T> | T>(url, data, config)
            .then((res) => unwrap<T>(res.data)),

    patch: <T, D = unknown>(url: string, data?: D, config?: AxiosRequestConfig) =>
        apiClient.patch<WrappedResponse<T> | T>(url, data, config)
            .then((res) => unwrap<T>(res.data)),

    delete: <T>(url: string, config?: AxiosRequestConfig) =>
        apiClient.delete<WrappedResponse<T> | T>(url, config)
            .then((res) => unwrap<T>(res.data)),
}
