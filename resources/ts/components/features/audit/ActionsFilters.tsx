import type { TFunction } from 'i18next'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ENTITIES, OPS, SOURCES, actionSourceLabel, entityLabel, type ActionFilters } from './actions'

const ALL = 'all'

const opName = (t: TFunction, op: (typeof OPS)[number]): string => ({
    create: t('settings:spaces.actions.filters.ops.create'),
    update: t('settings:spaces.actions.filters.ops.update'),
    delete: t('settings:spaces.actions.filters.ops.delete'),
    confirm: t('settings:spaces.actions.filters.ops.confirm'),
    skip: t('settings:spaces.actions.filters.ops.skip'),
    rollback: t('settings:spaces.actions.filters.ops.rollback'),
    cleanup: t('settings:spaces.actions.filters.ops.cleanup'),
} satisfies Record<(typeof OPS)[number], string>)[op]

interface ActionsFiltersProps {
    value: ActionFilters
    onChange: (next: ActionFilters) => void
}

/** Narrows the change log by operation, kind of entity and the door a change came through. */
export function ActionsFilters({ value, onChange }: ActionsFiltersProps) {
    const { t } = useTranslation('settings')
    const set = (key: keyof ActionFilters) => (v: string) => onChange({ ...value, [key]: v === ALL ? undefined : v })
    const active = !!(value.op || value.entityType || value.source)

    const pick = (key: keyof ActionFilters, label: string, items: { value: string; label: string }[]) => (
        <Select value={value[key] ?? ALL} onValueChange={set(key)}>
            <SelectTrigger className="w-full sm:w-44" aria-label={label}>
                <SelectValue />
            </SelectTrigger>
            <SelectContent>
                <SelectItem value={ALL}>
                    {label}: {t('spaces.actions.filters.all')}
                </SelectItem>
                {items.map((i) => (
                    <SelectItem key={i.value} value={i.value}>
                        {i.label}
                    </SelectItem>
                ))}
            </SelectContent>
        </Select>
    )

    return (
        <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
            {pick(
                'op',
                t('spaces.actions.filters.op'),
                OPS.map((o) => ({ value: o, label: opName(t, o) })),
            )}
            {pick(
                'entityType',
                t('spaces.actions.filters.entity'),
                ENTITIES.map((e) => ({ value: e, label: entityLabel(t, e) })),
            )}
            {pick(
                'source',
                t('spaces.actions.filters.source'),
                SOURCES.map((s) => ({ value: s, label: actionSourceLabel(t, s) })),
            )}
            {active && (
                <Button variant="ghost" onClick={() => onChange({})}>
                    {t('spaces.actions.filters.reset')}
                </Button>
            )}
        </div>
    )
}
