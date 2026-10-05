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
import type { Action, ActionType } from '@/types/automation'
import { ACTION_TYPES } from '@/types/automation'
import { AccountSelect, CategorySelect, TagSelect } from '@/components/shared'
import { actionTypeLabel } from '@/lib/labels'

interface ActionBuilderProps {
    value: Action[]
    onChange: (value: Action[]) => void
}

export function ActionBuilder({ value, onChange }: ActionBuilderProps) {
    const { t } = useTranslation('forms')

    const addAction = () => {
        onChange([
            ...value,
            { type: 'set_category' },
        ])
    }

    const updateAction = (index: number, updates: Partial<Action>) => {
        const newActions = value.map((action, i) =>
            i === index ? { ...action, ...updates } : action
        )
        onChange(newActions)
    }

    const removeAction = (index: number) => {
        onChange(value.filter((_, i) => i !== index))
    }

    const renderActionParams = (action: Action, index: number) => {
        switch (action.type) {
            case 'set_category':
                return (
                    <CategorySelect
                        plain
                        value={action.category_id as number | undefined}
                        onChange={(categoryId) => updateAction(index, { category_id: categoryId })}
                    />
                )

            case 'add_tags':
            case 'remove_tags':
                return (
                    <TagSelect
                        value={(action.tag_ids as number[]) || []}
                        onChange={(tagIds) => updateAction(index, { tag_ids: tagIds })}
                    />
                )

            case 'set_description':
                return (
                    <Input
                        className="w-full min-w-0"
                        value={(action.template || action.description || '') as string}
                        onChange={(e) => updateAction(index, { template: e.target.value })}
                        placeholder={t('automation.descriptionTemplate')}
                    />
                )

            case 'create_transfer':
                return (
                    <div className="grid grid-cols-2 gap-2">
                        <AccountSelect
                            plain
                            excludeDebts={false}
                            activeOnly={false}
                            value={action.to_account_id as number | undefined}
                            onChange={(accountId) => updateAction(index, { to_account_id: accountId })}
                            placeholder={t('automation.toAccount')}
                        />
                        <Input
                            className="min-w-0"
                            value={String(action.amount_formula ?? '')}
                            onChange={(e) => updateAction(index, { amount_formula: e.target.value })}
                            placeholder={t('automation.amountOrFormula')}
                        />
                    </div>
                )

            default:
                return null
        }
    }

    return (
        <div className="space-y-4">
            <div className="space-y-2">
                {value.map((action, index) => (
                    <div key={index} className="space-y-2 p-2 bg-muted/50 rounded">
                        <div className="flex items-center gap-2">
                            <Select
                                value={action.type}
                                onValueChange={(type) => updateAction(index, { type: type as ActionType })}
                            >
                                <SelectTrigger className="w-full">
                                    <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                    {ACTION_TYPES.map(actionType => (
                                        <SelectItem key={actionType} value={actionType}>
                                            {actionTypeLabel(t, actionType)}
                                        </SelectItem>
                                    ))}
                                </SelectContent>
                            </Select>

                            <Button
                                type="button"
                                variant="ghost"
                                size="icon"
                                className="shrink-0"
                                onClick={() => removeAction(index)}
                            >
                                <Trash2 className="size-4 text-muted-foreground hover:text-destructive" />
                            </Button>
                        </div>

                        {renderActionParams(action, index)}
                    </div>
                ))}
            </div>

            <Button type="button" variant="outline" size="sm" onClick={addAction}>
                <Plus className="size-4 mr-2" />
                {t('automation.addAction')}
            </Button>

            {value.length === 0 && (
                <p className="text-sm text-muted-foreground text-center py-4 border rounded-lg border-dashed">
                    {t('automation.noActions')}
                </p>
            )}
        </div>
    )
}
