import { useTranslation } from 'react-i18next'
import { Eye, Pencil, Trash2, type LucideIcon } from 'lucide-react'
import { Checkbox } from '@/components/ui/checkbox'
import {
    API_PERMISSION_LEVELS,
    permissionLevel,
    type ApiPermission,
    type ApiPermissionLevel,
    type ApiResource,
} from '@/types'
import { apiResourceLabel } from '@/lib/labels'

/** One row per part of a space: no access, read, read & write, or read, write & delete. */
export function PermissionGrid({
    value,
    onChange,
    resources,
    maxLevel = 'delete',
}: {
    value: ApiPermission[]
    onChange: (value: ApiPermission[]) => void
    /** Parts of a space on offer, from the server. */
    resources: readonly ApiResource[]
    /** Highest level on offer. */
    maxLevel?: ApiPermissionLevel
}) {
    const { t } = useTranslation('settings')
    const columnLabel: Record<ApiPermissionLevel, string> = {
        read: t('api.column.read'),
        write: t('api.column.write'),
        delete: t('api.column.delete'),
    }
    const icon: Record<ApiPermissionLevel, LucideIcon> = { read: Eye, write: Pencil, delete: Trash2 }
    const levels = API_PERMISSION_LEVELS.slice(0, API_PERMISSION_LEVELS.indexOf(maxLevel) + 1)

    // Each level includes the ones before it: ticking Write ticks Read, and
    // clearing Read clears Write and Delete.
    const toggle = (resource: ApiResource, level: ApiPermissionLevel, on: boolean) => {
        const rest = value.filter((p) => !p.startsWith(`${resource}:`))
        const next = on ? level : API_PERMISSION_LEVELS[API_PERMISSION_LEVELS.indexOf(level) - 1]
        onChange(next ? [...rest, `${resource}:${next}` as ApiPermission] : rest)
    }

    const columns = { gridTemplateColumns: `minmax(0,1fr) repeat(${levels.length}, 1.75rem)` }

    return (
        <div className="divide-y rounded-lg border">
            <div className="grid items-center gap-1 px-3 py-1 text-xs text-muted-foreground" style={columns}>
                <span />
                {levels.map((level) => {
                    const Icon = icon[level]
                    return (
                        <span
                            key={level}
                            className="flex justify-center"
                            title={columnLabel[level]}
                            aria-label={columnLabel[level]}
                        >
                            <Icon className="size-3.5" />
                        </span>
                    )
                })}
            </div>
            {resources.map((resource) => {
                const held = permissionLevel(value, resource)
                const rank = held ? API_PERMISSION_LEVELS.indexOf(held) : -1
                return (
                    <div key={resource} className="grid items-center gap-1 px-3 py-1" style={columns}>
                        <span className="min-w-0 truncate text-sm">{apiResourceLabel(t, resource)}</span>
                        {levels.map((level) => (
                            <span key={level} className="flex justify-center">
                                <Checkbox
                                    checked={API_PERMISSION_LEVELS.indexOf(level) <= rank}
                                    onCheckedChange={(on) => toggle(resource, level, on === true)}
                                    aria-label={`${apiResourceLabel(t, resource)}: ${columnLabel[level]}`}
                                />
                            </span>
                        ))}
                    </div>
                )
            })}
        </div>
    )
}
