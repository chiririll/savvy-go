import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import { Plus, Trash2 } from 'lucide-react'
import type { Condition, ConditionGroup, ConditionOperator } from '@/types/automation'
import { CONDITION_FIELDS } from '@/types/automation'
import { TRANSACTION_TYPES } from '@/constants'
import { AccountSelect, CategorySelect, SegmentedChoice, TagSelect } from '@/components/shared'
import { conditionFieldLabel, operatorLabel, transactionTypeLabel } from '@/lib/labels'

interface ConditionBuilderProps {
    value: ConditionGroup
    onChange: (value: ConditionGroup) => void
}

const OPERATORS: Record<string, ConditionOperator[]> = {
    equals: ['equals', 'not_equals'],
    in: ['in', 'not_in'],
    gt: ['gt', 'gte', 'lt', 'lte', 'between'],
    contains: ['contains', 'not_contains', 'starts_with', 'ends_with', 'matches'],
    is_null: ['is_null', 'is_not_null'],
    has_any: ['has_any', 'has_all', 'has_none'],
}

export function ConditionBuilder({ value, onChange }: ConditionBuilderProps) {
    const { t } = useTranslation(['forms', 'pages', 'common'])

    const addCondition = () => {
        onChange({
            ...value,
            conditions: [
                ...value.conditions,
                { field: 'type', op: 'equals', value: 'expense' },
            ],
        })
    }

    const updateCondition = (index: number, updates: Partial<Condition>) => {
        const newConditions = value.conditions.map((condition, i) =>
            i === index ? { ...condition, ...updates } : condition
        )
        onChange({ ...value, conditions: newConditions })
    }

    const removeCondition = (index: number) => {
        onChange({
            ...value,
            conditions: value.conditions.filter((_, i) => i !== index),
        })
    }

    const getOperatorsForField = (field: string): ConditionOperator[] => {
        const fieldConfig = CONDITION_FIELDS.find(f => f.value === field)
        if (!fieldConfig) return []

        return fieldConfig.operators.flatMap(op => OPERATORS[op] ?? [])
    }

    const renderValueInput = (condition: Condition, index: number) => {
        const { field, op } = condition

        if (op === 'is_null' || op === 'is_not_null') {
            return null
        }

        if (field === 'type') {
            if (op === 'in' || op === 'not_in') {
                return (
                    <div className="flex flex-wrap gap-1">
                        {TRANSACTION_TYPES.map(type => (
                            <Button
                                key={type}
                                type="button"
                                variant={(condition.value as string[])?.includes(type) ? 'default' : 'outline'}
                                size="sm"
                                onClick={() => {
                                    const current = (condition.value as string[]) || []
                                    const newValue = current.includes(type)
                                        ? current.filter(v => v !== type)
                                        : [...current, type]
                                    updateCondition(index, { value: newValue })
                                }}
                            >
                                {transactionTypeLabel(t, type)}
                            </Button>
                        ))}
                    </div>
                )
            }
            return (
                <Select
                    value={condition.value as string}
                    onValueChange={(val) => updateCondition(index, { value: val })}
                >
                    <SelectTrigger className="w-full min-w-0">
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                        {TRANSACTION_TYPES.map(type => (
                            <SelectItem key={type} value={type}>
                                {transactionTypeLabel(t, type)}
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>
            )
        }

        if (field === 'account_id') {
            return (
                <AccountSelect
                    plain
                    excludeDebts={false}
                    activeOnly={false}
                    value={condition.value as number | undefined}
                    onChange={(accountId) => updateCondition(index, { value: accountId })}
                />
            )
        }

        if (field === 'category_id') {
            return (
                <CategorySelect
                    plain
                    value={condition.value as number | undefined}
                    onChange={(categoryId) => updateCondition(index, { value: categoryId })}
                />
            )
        }

        if (field === 'tags') {
            return (
                <TagSelect
                    value={(condition.value as number[]) || []}
                    onChange={(tagIds) => updateCondition(index, { value: tagIds })}
                />
            )
        }

        if (field === 'amount') {
            if (op === 'between') {
                const [min, max] = (condition.value as [number, number]) || [0, 0]
                return (
                    <div className="flex items-center gap-2">
                        <Input
                            type="number"
                            className="min-w-0 flex-1"
                            value={min}
                            onChange={(e) => updateCondition(index, { value: [Number(e.target.value), max] })}
                        />
                        <span className="text-muted-foreground">{t('automation.and')}</span>
                        <Input
                            type="number"
                            className="min-w-0 flex-1"
                            value={max}
                            onChange={(e) => updateCondition(index, { value: [min, Number(e.target.value)] })}
                        />
                    </div>
                )
            }
            return (
                <Input
                    type="number"
                    className="w-full min-w-0"
                    value={condition.value as number}
                    onChange={(e) => updateCondition(index, { value: Number(e.target.value) })}
                />
            )
        }

        return (
            <Input
                className="w-full min-w-0"
                value={condition.value as string}
                onChange={(e) => updateCondition(index, { value: e.target.value })}
                placeholder={t('automation.enterValue')}
            />
        )
    }

    return (
        <div className="space-y-4">
            <SegmentedChoice
                value={value.match}
                onChange={(match) => onChange({ ...value, match })}
                options={[
                    { value: 'all' as const, label: t('forms:automation.matchAll') },
                    { value: 'any' as const, label: t('forms:automation.matchAny') },
                ]}
            />

            <div className="space-y-2">
                {value.conditions.map((condition, index) => {
                    const valueInput = renderValueInput(condition, index)

                    return (
                    <div key={index} className="space-y-2 p-2 bg-muted/50 rounded">
                        <div className="flex items-center gap-2">
                            <Select
                                value={condition.field}
                                onValueChange={(field) => {
                                    const operators = getOperatorsForField(field)
                                    updateCondition(index, {
                                        field,
                                        op: operators[0] || 'equals',
                                        value: null,
                                    })
                                }}
                            >
                                <SelectTrigger className="w-full min-w-0 flex-1">
                                    <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                    {CONDITION_FIELDS.map(field => (
                                        <SelectItem key={field.value} value={field.value}>
                                            {conditionFieldLabel(t, field.value)}
                                        </SelectItem>
                                    ))}
                                </SelectContent>
                            </Select>

                            <Select
                                value={condition.op}
                                onValueChange={(op) => updateCondition(index, { op: op as ConditionOperator })}
                            >
                                <SelectTrigger className="w-full min-w-0 flex-1">
                                    <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                    {getOperatorsForField(condition.field).map(op => (
                                        <SelectItem key={op} value={op}>
                                            {operatorLabel(t, op)}
                                        </SelectItem>
                                    ))}
                                </SelectContent>
                            </Select>

                            <Button
                                type="button"
                                variant="ghost"
                                size="icon"
                                className="shrink-0"
                                onClick={() => removeCondition(index)}
                            >
                                <Trash2 className="size-4 text-muted-foreground hover:text-destructive" />
                            </Button>
                        </div>

                        {valueInput}
                    </div>
                    )
                })}
            </div>

            <Button type="button" variant="outline" size="sm" onClick={addCondition}>
                <Plus className="size-4 mr-2" />
                {t('automation.addCondition')}
            </Button>
        </div>
    )
}
