import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { accountsApi, reconciliationsApi } from '@/api'
import type { ReconciliationInput } from '@/types'
import i18n from '@/lib/i18n'
import { getApiErrorMessage } from '@/lib/api-error'
import { numberToAmount } from '@/lib/decimal-amount'

const QUERY_KEY = ['reconciliations']

/** A point moves the balance of its account from its date on, so everything built on balances is stale after a write. */
const RECONCILIATION_INVALIDATE = [QUERY_KEY, ['accounts'], ['reports'], ['transactions'], ['budgets']] as const

/** Every reconciliation day, newest first. */
export function useAllReconciliationDays() {
    return useQuery({
        queryKey: [...QUERY_KEY, 'days', 'all'],
        queryFn: () => reconciliationsApi.getDays(),
    })
}

export function useAccountReconciliations(accountId: number | string, period?: { start_date?: string; end_date?: string }) {
    return useQuery({
        queryKey: [...QUERY_KEY, 'account', accountId, period],
        queryFn: () => reconciliationsApi.getForAccount(accountId, period),
        enabled: !!accountId,
    })
}

/** Points grouped by day, for the transaction list. */
export function useReconciliationDays(period: { start_date: string; end_date: string }, enabled = true) {
    return useQuery({
        queryKey: [...QUERY_KEY, 'days', period],
        queryFn: () => reconciliationsApi.getDays(period),
        enabled,
    })
}

function useInvalidateReconciliations() {
    const queryClient = useQueryClient()
    return () => RECONCILIATION_INVALIDATE.forEach((queryKey) => queryClient.invalidateQueries({ queryKey }))
}

export function useCreateReconciliations() {
    const invalidate = useInvalidateReconciliations()

    return useMutation({
        mutationFn: (input: ReconciliationInput) => reconciliationsApi.create(input),
        onSuccess: () => {
            invalidate()
            toast.success(i18n.t('toasts.reconciliation.created'))
        },
        onError: (error: unknown) => {
            toast.error(getApiErrorMessage(error, i18n.t('toasts.reconciliation.createFailed')))
        },
    })
}

export function useDeleteReconciliation() {
    const invalidate = useInvalidateReconciliations()

    return useMutation({
        mutationFn: (id: number | string) => reconciliationsApi.delete(id),
        onSuccess: () => {
            invalidate()
            toast.success(i18n.t('toasts.reconciliation.deleted'))
        },
        onError: (error: unknown) => {
            toast.error(getApiErrorMessage(error, i18n.t('toasts.reconciliation.deleteFailed')))
        },
    })
}

/**
 * The ledger's balance of each account at the end of a date, in the account's own currency, as exact text
 * (by account id). `decimals` gives the decimals of each account.
 */
export function useBalancesAt(date: string, decimals: ReadonlyMap<number, number>, enabled = true) {
    const ids = [...decimals.keys()].sort((a, b) => a - b)

    return useQuery({
        queryKey: ['accounts', 'balances-at', date, ids],
        queryFn: async () => {
            const history = await accountsApi.getBalanceHistory({ start_date: date, end_date: date, account_ids: ids })
            const balances = new Map<number, string>()
            for (const series of history.series) {
                const value = series.native_data?.[0]
                if (series.id !== null && value !== undefined) {
                    balances.set(series.id, numberToAmount(value, decimals.get(series.id) ?? 2))
                }
            }
            return balances
        },
        enabled: enabled && ids.length > 0 && date !== '',
    })
}
