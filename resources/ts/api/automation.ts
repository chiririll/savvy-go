import { api } from './client'
import { createCrudApi } from './crud'
import type { AutomationRule, AutomationRuleLog, TriggerType } from '@/types/automation'
import type { AutomationRuleFormData } from '@/schemas'

const ENDPOINT = '/automation-rules'
const crud = createCrudApi<AutomationRule, AutomationRuleFormData>(ENDPOINT)

export const automationApi = {
    ...crud,

    toggle: (id: number | string) =>
        api.post<AutomationRule>(`${ENDPOINT}/${id}/toggle`),

    reorder: (rules: Array<{ id: number; priority: number }>) =>
        api.post<{ success: boolean }, { rules: Array<{ id: number; priority: number }> }>(`${ENDPOINT}/reorder`, { rules }),

    test: (id: number | string, transactionId: number) =>
        api.post<{ conditionsMatch: boolean; actions: unknown[] }, { transaction_id: number }>(
            `${ENDPOINT}/${id}/test`,
            { transaction_id: transactionId }
        ),

    getLogs: (id: number | string) =>
        api.get<AutomationRuleLog[]>(`${ENDPOINT}/${id}/logs`),

    getTriggers: () =>
        api.get<TriggerType[]>(`${ENDPOINT}/triggers`),
}
