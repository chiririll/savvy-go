import { useTranslation } from 'react-i18next'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { API_RESOURCES, permissionLevel, type ApiPermission, type ApiPermissionLevel } from '@/types'
import { apiResourceLabel } from '@/lib/labels'

/** One row per part of a space: no access, read, or read & write. */
export function PermissionGrid({
    value,
    onChange,
}: {
    value: ApiPermission[]
    onChange: (value: ApiPermission[]) => void
}) {
    const { t } = useTranslation('settings')

    const set = (resource: (typeof API_RESOURCES)[number], level: ApiPermissionLevel | 'none') => {
        const rest = value.filter((p) => !p.startsWith(`${resource}:`))
        onChange(level === 'none' ? rest : [...rest, `${resource}:${level}` as ApiPermission])
    }

    return (
        <div className="divide-y rounded-lg border">
            {API_RESOURCES.map((resource) => (
                <div key={resource} className="flex items-center justify-between gap-3 px-3 py-1.5">
                    <span className="min-w-0 truncate text-sm">{apiResourceLabel(t, resource)}</span>
                    <Select
                        value={permissionLevel(value, resource) ?? 'none'}
                        onValueChange={(v) => set(resource, v as ApiPermissionLevel | 'none')}
                    >
                        <SelectTrigger size="sm" className="w-36" aria-label={apiResourceLabel(t, resource)}>
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="none">{t('api.level.none')}</SelectItem>
                            <SelectItem value="read">{t('api.level.read')}</SelectItem>
                            <SelectItem value="write">{t('api.level.write')}</SelectItem>
                        </SelectContent>
                    </Select>
                </div>
            ))}
        </div>
    )
}
