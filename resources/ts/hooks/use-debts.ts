import { useQuery } from '@tanstack/react-query'
import { debtsApi } from '@/api'
import { DebtFormData, DebtPaymentFormData } from '@/schemas'
import { useResourceItem, useResourceMutation } from './use-crud'
import i18n from '@/lib/i18n'

const QUERY_KEY = ['debts']
const DEBT_INVALIDATE = [QUERY_KEY, ['accounts']] as const
const DEBT_TX_INVALIDATE = [QUERY_KEY, ['accounts'], ['transactions']] as const

export function useDebts(params?: { include_completed?: boolean }) {
    return useQuery({
        queryKey: params ? [...QUERY_KEY, params] : QUERY_KEY,
        queryFn: () => debtsApi.getAll(params),
    })
}

export function useDebtsWithSummary(params?: { include_completed?: boolean }, { enabled = true }: { enabled?: boolean } = {}) {
    return useQuery({
        queryKey: [...QUERY_KEY, 'with-summary', params],
        queryFn: () => debtsApi.getAllWithSummary(params),
        enabled,
    })
}

export function useDebtSummary() {
    return useQuery({
        queryKey: [...QUERY_KEY, 'summary'],
        queryFn: () => debtsApi.getSummary(),
    })
}

export function useDebt(id: string | number) {
    return useResourceItem(QUERY_KEY, () => debtsApi.getById(id), id)
}

export function useCreateDebt(redirectTo?: string) {
    return useResourceMutation({
        mutationFn: (data: DebtFormData) => debtsApi.create(data),
        invalidateKeys: [...DEBT_TX_INVALIDATE],
        successMessage: i18n.t('toasts.debt.created'),
        redirectTo,
    })
}

export function useUpdateDebt(redirectTo?: string) {
    return useResourceMutation({
        mutationFn: ({ id, data }: { id: string | number; data: Partial<DebtFormData> }) =>
            debtsApi.update(id, data),
        invalidateKeys: [...DEBT_INVALIDATE],
        successMessage: i18n.t('toasts.debt.updated'),
        redirectTo,
    })
}

export function useDeleteDebt() {
    return useResourceMutation({
        mutationFn: (id: string | number) => debtsApi.delete(id),
        invalidateKeys: [...DEBT_INVALIDATE],
        successMessage: i18n.t('toasts.debt.deleted'),
    })
}

export function useDebtPayment() {
    return useResourceMutation({
        mutationFn: ({ debtId, data }: { debtId: string | number; data: DebtPaymentFormData }) =>
            debtsApi.makePayment(debtId, data),
        invalidateKeys: [...DEBT_TX_INVALIDATE],
        successMessage: i18n.t('toasts.debt.payment'),
    })
}

export function useDebtCollection() {
    return useResourceMutation({
        mutationFn: ({ debtId, data }: { debtId: string | number; data: DebtPaymentFormData }) =>
            debtsApi.collectPayment(debtId, data),
        invalidateKeys: [...DEBT_TX_INVALIDATE],
        successMessage: i18n.t('toasts.debt.collection'),
    })
}

export function useReopenDebt() {
    return useResourceMutation({
        mutationFn: (id: string | number) => debtsApi.reopen(id),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.debt.reopened'),
    })
}
