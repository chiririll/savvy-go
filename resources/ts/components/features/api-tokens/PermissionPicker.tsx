import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronRight } from 'lucide-react'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { cn } from '@/lib/utils'
import { testIdSpoiler } from '@/lib/test-id'
import {
    API_PERMISSION_LEVELS,
    samePermissions,
    type ApiPermission,
    type ApiPermissionLevel,
    type ApiTokenPreset,
    type KindOptions,
} from '@/types'
import { apiPresetHint, apiPresetLabel } from '@/lib/labels'
import { PermissionGrid } from './PermissionGrid'

type Tab = ApiTokenPreset | 'custom'

/**
 * What a token or app may do: a preset as a tab, with the rights it grants
 * shown below. Changing a right moves the tab to the preset that now matches,
 * or to Custom. `options` is what the server lets this kind of token hold (a
 * connected AI app gets only the parts its tools use); `maxLevel` narrows it
 * further, and presets that would exceed it are left out.
 */
export function PermissionPicker({
    value,
    onChange,
    options,
    maxLevel = 'delete',
}: {
    value: ApiPermission[]
    onChange: (value: ApiPermission[]) => void
    /** What this kind of token can be given, from the server. */
    options: KindOptions
    maxLevel?: ApiPermissionLevel
}) {
    const { t } = useTranslation('settings')
    // Custom was picked on purpose: stay on it even while the rights equal a preset.
    const [pinnedCustom, setPinnedCustom] = useState(false)
    const [details, setDetails] = useState(false)

    const rank = (level: ApiPermissionLevel) => API_PERMISSION_LEVELS.indexOf(level)
    const { resources } = options
    const presets = options.presets.filter((p) =>
        p.permissions.every((g) => rank(g.split(':')[1] as ApiPermissionLevel) <= rank(maxLevel)),
    )

    const matching = presets.find((p) => samePermissions(value, p.permissions, resources))?.name
    const tab: Tab = pinnedCustom || !matching ? 'custom' : matching

    const pick = (next: string) => {
        if (next === 'custom') {
            setPinnedCustom(true)
            return
        }
        setPinnedCustom(false)
        onChange(presets.find((p) => p.name === next)?.permissions ?? [])
    }

    return (
        <div className="space-y-3">
            <Tabs value={tab} onValueChange={pick}>
                <TabsList className="w-full">
                    {presets.map(({ name }) => (
                        <TabsTrigger key={name} value={name} aria-controls={undefined}>
                            {apiPresetLabel(t, name)}
                        </TabsTrigger>
                    ))}
                    <TabsTrigger value="custom" aria-controls={undefined}>{apiPresetLabel(t, 'custom')}</TabsTrigger>
                </TabsList>
            </Tabs>
            <p className="text-xs text-muted-foreground">{apiPresetHint(t, tab)}</p>
            <Collapsible open={details} onOpenChange={setDetails} className="space-y-3">
                <CollapsibleTrigger
                    className="flex items-center gap-1 text-xs font-medium text-muted-foreground hover:text-foreground"
                    {...testIdSpoiler()}
                >
                    <ChevronRight className={cn('size-3.5 transition-transform', details && 'rotate-90')} />
                    {t('api.create.permissionsDetails')}
                </CollapsibleTrigger>
                <CollapsibleContent>
                    <PermissionGrid
                        value={value}
                        resources={resources}
                        maxLevel={maxLevel}
                        onChange={(next) => {
                            setPinnedCustom(false)
                            onChange(next)
                        }}
                    />
                </CollapsibleContent>
            </Collapsible>
        </div>
    )
}
