import { create } from 'zustand'
import { authApi } from '@/api'
import { queryClient } from '@/lib/query-client'
import { useAuthStore } from '@/stores/auth'
import type { ApiError, ConfirmFields } from '@/types'

/** What the dialog asks for: the password, or with 2FA on an authenticator or recovery code. */
export type ConfirmMode = 'password' | 'code'

/** The user closed the dialog. Not a failure: the change simply did not happen. */
export class ConfirmCancelled extends Error {
    constructor() {
        super('confirmation cancelled')
    }
}

export const isConfirmCancelled = (error: unknown): boolean => error instanceof ConfirmCancelled

interface Pending {
    mode: ConfirmMode
    run: (fields: ConfirmFields) => Promise<unknown>
    resolve: (value: unknown) => void
    reject: (reason: unknown) => void
}

interface ConfirmState {
    pending: Pending | null
}

export const useConfirmStore = create<ConfirmState>(() => ({ pending: null }))

/**
 * What proves who the user is. The server decides the same way: 2FA means a code,
 * otherwise the password, and an account with neither (single sign-on only) has
 * nothing to confirm with.
 */
async function confirmMode(): Promise<ConfirmMode | null> {
    const status = await queryClient.fetchQuery({
        queryKey: ['two-factor-status'],
        queryFn: authApi.twoFactorStatus,
        staleTime: 0,
    })
    if (status.enabled) return 'code'
    return useAuthStore.getState().user?.isSsoOnly ? null : 'password'
}

/**
 * Runs a change to the account's credentials behind a dialog that asks for the
 * password or code. `run` gets the fields to send with the request; a wrong answer
 * keeps the dialog open for another try. Rejects with `ConfirmCancelled` when the
 * user closes the dialog.
 */
export async function withConfirmation<T>(run: (fields: ConfirmFields) => Promise<T>): Promise<T> {
    const mode = await confirmMode()
    if (mode === null) return run({})
    if (useConfirmStore.getState().pending) throw new ConfirmCancelled()
    return new Promise<T>((resolve, reject) => {
        useConfirmStore.setState({
            pending: { mode, run, resolve: resolve as (value: unknown) => void, reject },
        })
    })
}

/** The server said the answer was wrong, or the account has had too many tries. */
function staysOpen(error: unknown): boolean {
    const e = error as Partial<ApiError> | null
    return e?.reason === 'confirmation_invalid' || e?.code === '429'
}

/**
 * Sends the change with what the user typed. Returns the message to show when the
 * dialog stays open for another try, and null when it is done.
 */
export async function submitConfirmation(fields: ConfirmFields): Promise<string | null> {
    const pending = useConfirmStore.getState().pending
    if (!pending) return null
    try {
        const result = await pending.run(fields)
        useConfirmStore.setState({ pending: null })
        pending.resolve(result)
        return null
    } catch (error) {
        if (staysOpen(error)) return (error as ApiError).message
        useConfirmStore.setState({ pending: null })
        pending.reject(error)
        return null
    }
}

export function cancelConfirmation(): void {
    const pending = useConfirmStore.getState().pending
    if (!pending) return
    useConfirmStore.setState({ pending: null })
    pending.reject(new ConfirmCancelled())
}
