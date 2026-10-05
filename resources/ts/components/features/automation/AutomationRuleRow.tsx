import { History } from 'lucide-react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Checkbox } from '@/components/ui/checkbox'
import { DropdownMenuItem } from '@/components/ui/dropdown-menu'
import { FeedRow, FeedStatusBadge, RowActions } from '@/components/shared'
import type { AutomationRule } from '@/types/automation'
import { triggerLabel } from '@/lib/labels'

interface AutomationRuleRowProps {
    rule: AutomationRule
    onEdit?: (rule: AutomationRule) => void
    onDelete?: (id: number) => void
    onToggle?: (id: number) => void
    isReadOnly?: boolean
}

export function AutomationRuleRow({ rule, onEdit, onDelete, onToggle, isReadOnly }: AutomationRuleRowProps) {
    const { t } = useTranslation(['common', 'pages', 'forms'])
    const canEdit = !!onEdit && !isReadOnly
    const canDelete = !!onDelete && !isReadOnly
    const counts = { conditions: rule.conditions.conditions.length, actions: rule.actions.length }

    return (
        <FeedRow
            // The active toggle sits where a transaction shows its icon.
            icon={(
                <label
                    className="flex size-full cursor-pointer items-center justify-center"
                    onClick={(event) => event.stopPropagation()}
                    onKeyDown={(event) => event.stopPropagation()}
                >
                    <Checkbox
                        className="size-5"
                        checked={rule.isActive}
                        disabled={isReadOnly || !onToggle}
                        onCheckedChange={() => onToggle?.(rule.id)}
                        aria-label={t('pages:automation.active')}
                    />
                </label>
            )}
            title={rule.name}
            badge={<FeedStatusBadge variant="outline">#{rule.priority}</FeedStatusBadge>}
            subtitle={triggerLabel(t, rule.triggerType)}
            amount={t('pages:automation.runs', { count: rule.runsCount })}
            amountClassName="font-sans font-medium"
            extraAmount={(
                <span title={t('pages:automation.shapeTitle', counts)}>
                    {t('pages:automation.shape', counts)}
                </span>
            )}
            onOpen={canEdit ? () => onEdit(rule) : undefined}
            hasActions
            actions={({ menuOpen, setMenuOpen, isMobile }) => (
                <RowActions
                    open={menuOpen}
                    onOpenChange={setMenuOpen}
                    showTrigger={!isMobile}
                    onEdit={canEdit ? () => onEdit(rule) : undefined}
                    onDelete={canDelete ? () => onDelete(rule.id) : undefined}
                    deleteTitle={t('pages:automation.deleteTitle')}
                    deleteDescription={t('pages:automation.deleteDescription')}
                >
                    <DropdownMenuItem asChild>
                        <Link to={`/automation/${rule.id}/logs`}>
                            <History className="mr-2 size-4" />
                            {t('actions.viewLogs')}
                        </Link>
                    </DropdownMenuItem>
                </RowActions>
            )}
        />
    )
}
