import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Plus } from 'lucide-react'
import type { Condition, ConditionGroup } from '@/types/automation'
import { SegmentedChoice } from '@/components/shared'
import { ConditionRow } from './ConditionRow'

interface ConditionBuilderProps {
    value: ConditionGroup
    onChange: (value: ConditionGroup) => void
}

export function ConditionBuilder({ value, onChange }: ConditionBuilderProps) {
    const { t } = useTranslation(['forms', 'pages', 'common'])

    const addCondition = () => {
        onChange({
            ...value,
            conditions: [...value.conditions, { field: 'type', op: 'equals', value: 'expense' }],
        })
    }

    const updateCondition = (index: number, updates: Partial<Condition>) => {
        onChange({
            ...value,
            conditions: value.conditions.map((condition, i) =>
                i === index ? { ...condition, ...updates } : condition,
            ),
        })
    }

    const removeCondition = (index: number) => {
        onChange({ ...value, conditions: value.conditions.filter((_, i) => i !== index) })
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
                {value.conditions.map((condition, index) => (
                    <ConditionRow
                        key={index}
                        condition={condition}
                        onChange={(updates) => updateCondition(index, updates)}
                        onRemove={() => removeCondition(index)}
                    />
                ))}
            </div>

            <Button type="button" variant="outline" size="sm" onClick={addCondition}>
                <Plus className="size-4 mr-2" />
                {t('automation.addCondition')}
            </Button>
        </div>
    )
}
