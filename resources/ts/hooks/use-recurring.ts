import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { recurringApi } from '@/api'
import type { RecurringInput } from '@/types'
import { useResourceItem, useResourceMutation } from './use-crud'
import i18n from '@/lib/i18n'
import { isApproximateRule } from '@/lib/recurring-template'

const QUERY_KEY = ['recurring']

export function useRecurring() {
    return useQuery({
        queryKey: QUERY_KEY,
        queryFn: () => recurringApi.getAll(),
    })
}

/** The ids of the rules whose amount is a guess: a pending occurrence of one shows `≈`. */
export function useApproximateRuleIds(): ReadonlySet<number> {
    const { data } = useRecurring()
    return useMemo(
        () => new Set((data ?? []).filter((rule) => isApproximateRule(rule.template)).map((rule) => rule.id)),
        [data],
    )
}

export function useRecurringById(id: string | number) {
    return useResourceItem(QUERY_KEY, () => recurringApi.getById(id), id)
}

export function useUpcomingRecurring() {
    return useQuery({
        queryKey: [...QUERY_KEY, 'upcoming'],
        queryFn: () => recurringApi.getUpcoming(),
    })
}

export function useCreateRecurring(redirectTo?: string) {
    return useResourceMutation({
        mutationFn: (data: RecurringInput) => recurringApi.create(data),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.recurring.created'),
        redirectTo,
    })
}

export function useUpdateRecurring(redirectTo?: string) {
    return useResourceMutation({
        mutationFn: ({ id, data }: { id: string | number; data: Partial<RecurringInput> }) =>
            recurringApi.update(id, data),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.recurring.updated'),
        redirectTo,
    })
}

export function useDeleteRecurring() {
    return useResourceMutation({
        mutationFn: (id: string | number) => recurringApi.delete(id),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.recurring.deleted'),
    })
}
