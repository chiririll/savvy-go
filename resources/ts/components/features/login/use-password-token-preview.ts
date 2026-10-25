import { useEffect, useState } from 'react'
import { authApi } from '@/api'
import type { PasswordTokenPreview } from '@/types/users'

/** Looks up who a password link is for; `invalid` once the link is missing, expired or used. */
export function usePasswordTokenPreview(token: string | undefined) {
    const [preview, setPreview] = useState<PasswordTokenPreview | null>(null)
    const [loading, setLoading] = useState(true)
    const [invalid, setInvalid] = useState(false)

    useEffect(() => {
        if (!token) {
            setInvalid(true)
            setLoading(false)
            return
        }

        authApi.previewPasswordToken(token)
            .then(setPreview)
            .catch(() => setInvalid(true))
            .finally(() => setLoading(false))
    }, [token])

    return { preview, loading, invalid }
}
