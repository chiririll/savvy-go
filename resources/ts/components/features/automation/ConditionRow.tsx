import { useTranslation } from 'react-i18next'
import { Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import type { Condition, ConditionOperator } from '@/types/automation'
import { CONDITION_FIELDS } from '@/types/automation'
import { conditionFieldLabel, operatorLabel } from '@/lib/labels'
import { operatorsForField } from './condition-operators'
import { ConditionValueInput } from './ConditionValueInput'

interface ConditionRowProps {
    condition: Condition
    onChange: (updates: Partial<Condition>) => void
    onRemove: () => void
}

export function ConditionRow({ condition, onChange, onRemove }: ConditionRowProps) {
    const { t } = useTranslation(['forms', 'pages', 'common'])

    return (
        <div className="space-y-2 p-2 bg-muted/50 rounded">
            <div className="flex items-center gap-2">
                <Select
                    value={condition.field}
                    onValueChange={(field) =>
                        onChange({
                            field,
                            op: operatorsForField(field)[0] || 'equals',
                            value: null,
                        })
                    }
                >
                    <SelectTrigger className="w-full min-w-0 flex-1">
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                        {CONDITION_FIELDS.map((field) => (
                            <SelectItem key={field.value} value={field.value}>
                                {conditionFieldLabel(t, field.value)}
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>

                <Select value={condition.op} onValueChange={(op) => onChange({ op: op as ConditionOperator })}>
                    <SelectTrigger className="w-full min-w-0 flex-1">
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                        {operatorsForField(condition.field).map((op) => (
                            <SelectItem key={op} value={op}>
                                {operatorLabel(t, op)}
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>

                <Button type="button" variant="ghost" size="icon" className="shrink-0" onClick={onRemove}>
                    <Trash2 className="size-4 text-muted-foreground hover:text-destructive" />
                </Button>
            </div>

            <ConditionValueInput condition={condition} onChange={onChange} />
        </div>
    )
}
