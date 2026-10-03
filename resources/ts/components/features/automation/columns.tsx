import { ColumnDef } from '@tanstack/react-table'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { DropdownMenuItem } from '@/components/ui/dropdown-menu'
import { History } from 'lucide-react'
import { Link } from 'react-router-dom'
import { RowActions } from '@/components/shared'
import type { AutomationRule } from '@/types/automation'
import i18n from '@/lib/i18n'

interface ColumnOptions {
    onDelete: (id: number) => void
    onToggle: (id: number) => void
    onEdit: (rule: AutomationRule) => void
    isReadOnly?: boolean
}

export function createAutomationColumns({ onDelete, onToggle, onEdit, isReadOnly }: ColumnOptions): ColumnDef<AutomationRule>[] {
    return [
        {
            accessorKey: 'priority',
            header: () => i18n.t('pages:automation.columns.order'),
            cell: ({ row }) => (
                <span className="text-muted-foreground text-sm">{row.original.priority}</span>
            ),
            size: 50,
        },
        {
            accessorKey: 'name',
            header: () => i18n.t('pages:automation.columns.name'),
            cell: ({ row }) => (
                <div>
                    <div className="font-medium">{row.original.name}</div>
                    {row.original.description && (
                        <div className="text-sm text-muted-foreground truncate max-w-xs">
                            {row.original.description}
                        </div>
                    )}
                </div>
            ),
        },
        {
            accessorKey: 'trigger_type',
            header: () => i18n.t('pages:automation.columns.trigger'),
            cell: ({ row }) => (
                <Badge variant="outline">{i18n.t(`forms:automation.triggers.${row.original.triggerType}`)}</Badge>
            ),
        },
        {
            accessorKey: 'conditions',
            header: () => i18n.t('pages:automation.columns.conditions'),
            cell: ({ row }) => (
                <span className="text-sm text-muted-foreground">
                    {i18n.t('pages:automation.conditionCount', { count: row.original.conditions.conditions.length })}
                </span>
            ),
        },
        {
            accessorKey: 'actions',
            header: () => i18n.t('pages:automation.columns.actions'),
            cell: ({ row }) => (
                <span className="text-sm text-muted-foreground">
                    {i18n.t('pages:automation.actionCount', { count: row.original.actions.length })}
                </span>
            ),
        },
        {
            accessorKey: 'runs_count',
            header: () => i18n.t('pages:automation.columns.runs'),
            cell: ({ row }) => (
                <span className="text-sm">{row.original.runsCount}</span>
            ),
        },
        {
            accessorKey: 'is_active',
            header: () => i18n.t('pages:automation.columns.active'),
            cell: ({ row }) => (
                <Switch
                    checked={row.original.isActive}
                    disabled={isReadOnly}
                    onCheckedChange={() => onToggle(row.original.id)}
                />
            ),
        },
        {
            id: 'actions',
            cell: ({ row }) => (
                <RowActions
                    onEdit={() => onEdit(row.original)}
                    onDelete={() => onDelete(row.original.id)}
                    deleteTitle={i18n.t('pages:automation.deleteTitle')}
                    deleteDescription={i18n.t('pages:automation.deleteDescription')}
                    isReadOnly={isReadOnly}
                >
                    <DropdownMenuItem asChild>
                        <Link to={`/automation/${row.original.id}/logs`}>
                            <History className="size-4 mr-2" />
                            {i18n.t('actions.viewLogs')}
                        </Link>
                    </DropdownMenuItem>
                </RowActions>
            ),
        },
    ]
}
