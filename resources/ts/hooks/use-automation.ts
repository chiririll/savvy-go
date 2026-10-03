import { useQuery } from '@tanstack/react-query'
import { automationApi } from '@/api/automation'
import type { AutomationRule } from '@/types/automation'
import type { AutomationRuleFormData } from '@/schemas'
import { useResourceItem, useResourceMutation } from './use-crud'
import i18n from '@/lib/i18n'

const QUERY_KEY = ['automation-rules']

export function useAutomationRules() {
    return useQuery({
        queryKey: QUERY_KEY,
        queryFn: () => automationApi.getAll(),
    })
}

export function useAutomationRuleById(id: string | number) {
    return useResourceItem(QUERY_KEY, () => automationApi.getById(id), id)
}

export function useAutomationTriggers() {
    return useQuery({
        queryKey: [...QUERY_KEY, 'triggers'],
        queryFn: () => automationApi.getTriggers(),
        staleTime: Infinity,
    })
}

export function useAutomationRuleLogs(id: string | number) {
    return useQuery({
        queryKey: [...QUERY_KEY, id, 'logs'],
        queryFn: () => automationApi.getLogs(id),
        enabled: !!id,
    })
}

export function useCreateAutomationRule(redirectTo?: string) {
    return useResourceMutation({
        mutationFn: (data: AutomationRuleFormData) => automationApi.create(data),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.automation.created'),
        redirectTo,
    })
}

export function useUpdateAutomationRule(redirectTo?: string) {
    return useResourceMutation({
        mutationFn: ({ id, data }: { id: string | number; data: Partial<AutomationRuleFormData> }) =>
            automationApi.update(id, data),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.automation.updated'),
        redirectTo,
    })
}

export function useDeleteAutomationRule() {
    return useResourceMutation({
        mutationFn: (id: string | number) => automationApi.delete(id),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.automation.deleted'),
    })
}

export function useToggleAutomationRule() {
    return useResourceMutation({
        mutationFn: (id: string | number) => automationApi.toggle(id),
        invalidateKeys: [QUERY_KEY],
        successMessage: (data: AutomationRule) =>
            data.isActive
                ? i18n.t('toasts.automation.enabled')
                : i18n.t('toasts.automation.disabled'),
    })
}

export function useReorderAutomationRules() {
    return useResourceMutation({
        mutationFn: (rules: Array<{ id: number; priority: number }>) => automationApi.reorder(rules),
        invalidateKeys: [QUERY_KEY],
        successMessage: i18n.t('toasts.automation.reordered'),
    })
}

export function useTestAutomationRule() {
    return useResourceMutation({
        mutationFn: ({ ruleId, transactionId }: { ruleId: string | number; transactionId: number }) =>
            automationApi.test(ruleId, transactionId),
    })
}
