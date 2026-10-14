import { useTranslation } from 'react-i18next'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'

export interface SpaceChoice {
    all: boolean
    ids: number[]
}

/** Whether a choice names any space at all; a token or app with none is refused. */
export const hasSpaces = (choice: SpaceChoice) => choice.all || choice.ids.length > 0

/** Which spaces an app or token may reach: all of them, or a ticked few. */
export function SpacePicker({
    spaces,
    value,
    onChange,
    idPrefix,
}: {
    spaces: ReadonlyArray<{ id: number; name: string }>
    value: SpaceChoice
    onChange: (value: SpaceChoice) => void
    idPrefix: string
}) {
    const { t } = useTranslation('settings')
    const toggle = (id: number, on: boolean) =>
        onChange({ all: false, ids: on ? [...value.ids, id] : value.ids.filter((x) => x !== id) })

    return (
        <div className="space-y-2">
            <div className="max-h-40 space-y-1 overflow-y-auto rounded-lg border p-2">
                {spaces.map((space) => (
                    <div key={space.id} className="flex items-center gap-2 px-1 py-1">
                        <Checkbox
                            id={`${idPrefix}-${space.id}`}
                            checked={value.all || value.ids.includes(space.id)}
                            disabled={value.all}
                            onCheckedChange={(on) => toggle(space.id, on === true)}
                        />
                        <Label htmlFor={`${idPrefix}-${space.id}`} className="min-w-0 flex-1 truncate font-normal">
                            {space.name}
                        </Label>
                    </div>
                ))}
            </div>
            <div className="flex items-start gap-2">
                <Switch
                    id={`${idPrefix}-all`}
                    checked={value.all}
                    onCheckedChange={(on) => onChange({ all: on, ids: [] })}
                />
                <div className="space-y-0.5">
                    <Label htmlFor={`${idPrefix}-all`}>{t('api.spaces.all')}</Label>
                    <p className="text-xs text-muted-foreground">{t('api.spaces.allHint')}</p>
                </div>
            </div>
        </div>
    )
}
