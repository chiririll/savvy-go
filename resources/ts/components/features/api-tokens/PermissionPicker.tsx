import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
    API_PERMISSION_LEVELS,
    API_RESOURCES,
    API_TOKEN_PRESETS,
    presetPermissions,
    samePermissions,
    type ApiPermission,
    type ApiPermissionLevel,
    type ApiResource,
    type ApiTokenPreset,
} from '@/types'
import { apiPresetHint, apiPresetLabel } from '@/lib/labels'
import { PermissionGrid } from './PermissionGrid'

type Tab = ApiTokenPreset | 'custom'

/**
 * What a token or app may do: a preset as a tab, with the rights it grants
 * shown below. Changing a right moves the tab to the preset that now matches,
 * or to Custom. `resources` and `maxLevel` narrow what is on offer (an MCP app
 * only reads), and presets that would exceed them are left out.
 */
export function PermissionPicker({
    value,
    onChange,
    resources = API_RESOURCES,
    maxLevel = 'delete',
}: {
    value: ApiPermission[]
    onChange: (value: ApiPermission[]) => void
    resources?: readonly ApiResource[]
    maxLevel?: ApiPermissionLevel
}) {
    const { t } = useTranslation('settings')
    // Custom was picked on purpose: stay on it even while the rights equal a preset.
    const [pinnedCustom, setPinnedCustom] = useState(false)

    const rank = (level: ApiPermissionLevel) => API_PERMISSION_LEVELS.indexOf(level)
    const presets = API_TOKEN_PRESETS.filter((p) =>
        presetPermissions(p, resources).every((g) => rank(g.split(':')[1] as ApiPermissionLevel) <= rank(maxLevel)),
    )

    const matching = presets.find((p) => samePermissions(value, presetPermissions(p, resources)))
    const tab: Tab = pinnedCustom || !matching ? 'custom' : matching

    const pick = (next: string) => {
        if (next === 'custom') {
            setPinnedCustom(true)
            return
        }
        setPinnedCustom(false)
        onChange(presetPermissions(next as ApiTokenPreset, resources))
    }

    return (
        <div className="space-y-3">
            <Tabs value={tab} onValueChange={pick}>
                <TabsList className="w-full">
                    {presets.map((preset) => (
                        <TabsTrigger key={preset} value={preset}>
                            {apiPresetLabel(t, preset)}
                        </TabsTrigger>
                    ))}
                    <TabsTrigger value="custom">{apiPresetLabel(t, 'custom')}</TabsTrigger>
                </TabsList>
            </Tabs>
            <p className="text-xs text-muted-foreground">{apiPresetHint(t, tab)}</p>
            <PermissionGrid
                value={value}
                resources={resources}
                maxLevel={maxLevel}
                onChange={(next) => {
                    setPinnedCustom(false)
                    onChange(next)
                }}
            />
        </div>
    )
}
