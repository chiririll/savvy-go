// Sample automation data for the stories of this folder.
import { reply } from '@/test/api-mocks'
import { accounts, categories, tags } from '@/test/fixtures'
import type { AutomationRule } from '@/types/automation'

export const rule = (overrides: Partial<AutomationRule> = {}): AutomationRule => ({
    id: 1,
    name: 'Groceries from the shop',
    description: 'Puts the weekly shop in Groceries',
    triggerType: 'on_transaction_create',
    priority: 10,
    conditions: {
        match: 'all',
        conditions: [
            { field: 'type', op: 'equals', value: 'expense' },
            { field: 'description', op: 'contains', value: 'shop' },
        ],
    },
    actions: [{ type: 'set_category', category_id: 1 }],
    isActive: true,
    stopProcessing: false,
    runsCount: 42,
    lastRunAt: '2030-01-05T10:00:00Z',
    createdAt: '2030-01-01T00:00:00Z',
    updatedAt: '2030-01-01T00:00:00Z',
    ...overrides,
})

/** What the pickers in conditions and actions read. */
export const pickerApi = () => [
    reply('get', '/accounts', accounts),
    reply('get', '/categories', categories),
    reply('get', '/tags', tags),
    reply('get', '/automation-rules/triggers', ['on_transaction_create', 'on_transaction_update']),
]
