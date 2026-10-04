export type TriggerType =
    | 'on_transaction_create'
    | 'on_transaction_update'

export type ConditionOperator =
    | 'equals'
    | 'not_equals'
    | 'in'
    | 'not_in'
    | 'gt'
    | 'gte'
    | 'lt'
    | 'lte'
    | 'between'
    | 'contains'
    | 'not_contains'
    | 'starts_with'
    | 'ends_with'
    | 'matches'
    | 'is_null'
    | 'is_not_null'
    | 'has_any'
    | 'has_all'
    | 'has_none'

export type ActionType =
    | 'set_category'
    | 'add_tags'
    | 'remove_tags'
    | 'set_description'
    | 'create_transfer'

export interface Condition {
    field: string
    op: ConditionOperator
    value: unknown
}

export interface ConditionGroup {
    match: 'all' | 'any'
    conditions: Condition[]
}

export interface Action {
    type: ActionType
    [key: string]: unknown
}

export interface AutomationRule {
    id: number
    name: string
    description: string | null
    triggerType: TriggerType
    priority: number
    conditions: ConditionGroup
    actions: Action[]
    isActive: boolean
    stopProcessing: boolean
    runsCount: number
    lastRunAt: string | null
    createdAt: string | null
    updatedAt: string | null
}

export interface AutomationRuleLog {
    id: number
    ruleId: number
    triggerEntityType: string | null
    triggerEntityId: number | null
    actionsExecuted: Array<{ type: string; result: unknown }> | null
    status: 'success' | 'error' | 'skipped'
    errorMessage: string | null
    createdAt: string
}

// Labels are localized by value (automation.fields.*, automation.actionTypes.*).
export const CONDITION_FIELDS = [
    { value: 'type', operators: ['equals', 'in'] },
    { value: 'amount', operators: ['equals', 'gt', 'gte', 'lt', 'lte', 'between'] },
    { value: 'description', operators: ['contains', 'not_contains', 'starts_with', 'ends_with', 'matches'] },
    { value: 'account_id', operators: ['equals', 'in'] },
    { value: 'category_id', operators: ['equals', 'in', 'is_null', 'is_not_null'] },
    { value: 'tags', operators: ['has_any', 'has_all', 'has_none'] },
] as const

export const ACTION_TYPES: readonly ActionType[] = [
    'set_category',
    'add_tags',
    'remove_tags',
    'set_description',
    'create_transfer',
]
