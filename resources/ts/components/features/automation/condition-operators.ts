import type { ConditionOperator } from '@/types/automation'
import { CONDITION_FIELDS } from '@/types/automation'

const OPERATOR_GROUPS: Record<string, ConditionOperator[]> = {
    equals: ['equals', 'not_equals'],
    in: ['in', 'not_in'],
    gt: ['gt', 'gte', 'lt', 'lte', 'between'],
    contains: ['contains', 'not_contains', 'starts_with', 'ends_with', 'matches'],
    is_null: ['is_null', 'is_not_null'],
    has_any: ['has_any', 'has_all', 'has_none'],
}

export function operatorsForField(field: string): ConditionOperator[] {
    const config = CONDITION_FIELDS.find((item) => item.value === field)
    if (!config) return []

    return config.operators.flatMap((group) => OPERATOR_GROUPS[group] ?? [])
}
