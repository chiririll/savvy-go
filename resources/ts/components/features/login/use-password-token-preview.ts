import { useEffect, useState } from 'react'
import { authApi } from '@/api'
import { isForbiddenError } from '@/lib/api/api-error'
import type { PasswordTokenPreview } from '@/types/users'

/**
 * `invalid`: the link is missing, expired or used. `foreign`: it was sent to another account
 * than the one signed in here.
 */
export type PasswordTokenStatus = 'loading' | 'ready' | 'invalid' | 'foreign'

/** Looks up who a password link is for. */
export function usePasswordTokenPreview(token: string | undefined) {
    const [preview, setPreview] = useState<PasswordTokenPreview | null>(null)
    const [status, setStatus] = useState<PasswordTokenStatus>('loading')

    useEffect(() => {
        if (!token) {
            setStatus('invalid')
            return
        }

        authApi.previewPasswordToken(token)
            .then((found) => {
                setPreview(found)
                setStatus('ready')
            })
            .catch((error: unknown) => setStatus(isForbiddenError(error) ? 'foreign' : 'invalid'))
    }, [token])

    return { preview, status }
}
