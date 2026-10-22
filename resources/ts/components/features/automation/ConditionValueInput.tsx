import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import type { Condition } from '@/types/automation'
import { TRANSACTION_TYPES } from '@/constants'
import { AccountSelect } from '@/components/shared/pickers/AccountSelect'
import { CategorySelect, TagSelect } from '@/components/shared'
import { transactionTypeLabel } from '@/lib/labels'

interface ConditionValueInputProps {
    condition: Condition
    onChange: (updates: Partial<Condition>) => void
}

/** The value editor suited to the condition's field and operator; none for the null checks. */
export function ConditionValueInput({ condition, onChange }: ConditionValueInputProps) {
    const { t } = useTranslation(['forms', 'pages', 'common'])
    const { field, op } = condition
    const setValue = (value: Condition['value']) => onChange({ value })

    if (op === 'is_null' || op === 'is_not_null') return null

    if (field === 'type') {
        if (op === 'in' || op === 'not_in') {
            const current = (condition.value as string[]) || []
            return (
                <div className="flex flex-wrap gap-1">
                    {TRANSACTION_TYPES.map((type) => (
                        <Button
                            key={type}
                            type="button"
                            variant={current.includes(type) ? 'default' : 'outline'}
                            size="sm"
                            onClick={() =>
                                setValue(
                                    current.includes(type) ? current.filter((v) => v !== type) : [...current, type],
                                )
                            }
                        >
                            {transactionTypeLabel(t, type)}
                        </Button>
                    ))}
                </div>
            )
        }
        return (
            <Select value={condition.value as string} onValueChange={setValue}>
                <SelectTrigger className="w-full min-w-0">
                    <SelectValue />
                </SelectTrigger>
                <SelectContent>
                    {TRANSACTION_TYPES.map((type) => (
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
                onChange={setValue}
            />
        )
    }

    if (field === 'category_id') {
        return <CategorySelect plain value={condition.value as number | undefined} onChange={setValue} />
    }

    if (field === 'tags') {
        return <TagSelect value={(condition.value as number[]) || []} onChange={setValue} />
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
                        onChange={(e) => setValue([Number(e.target.value), max])}
                    />
                    <span className="text-muted-foreground">{t('automation.and')}</span>
                    <Input
                        type="number"
                        className="min-w-0 flex-1"
                        value={max}
                        onChange={(e) => setValue([min, Number(e.target.value)])}
                    />
                </div>
            )
        }
        return (
            <Input
                type="number"
                className="w-full min-w-0"
                value={condition.value as number}
                onChange={(e) => setValue(Number(e.target.value))}
            />
        )
    }

    return (
        <Input
            className="w-full min-w-0"
            value={condition.value as string}
            onChange={(e) => setValue(e.target.value)}
            placeholder={t('automation.enterValue')}
        />
    )
}
