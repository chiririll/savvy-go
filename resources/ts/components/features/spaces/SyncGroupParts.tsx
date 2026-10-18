import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Trash2 } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { spaceRoleLabel, syncKindLabel } from '@/lib/labels'
import type { Space, SyncChange, SyncGroup, SyncKind } from '@/types/spaces'

export const SYNC_KIND_LIST: SyncKind[] = ['categories', 'tags']

interface SyncKindSwitchesProps {
    idPrefix: string
    values: Record<SyncKind, boolean>
    disabled?: boolean
    onChange: (kind: SyncKind, on: boolean) => void
}

export function SyncKindSwitches({ idPrefix, values, disabled, onChange }: SyncKindSwitchesProps) {
    const { t } = useTranslation('settings')

    return (
        <div className="space-y-2">
            {SYNC_KIND_LIST.map((kind) => (
                <div key={kind} className="flex items-center justify-between gap-3 text-sm">
                    <Label htmlFor={`${idPrefix}-${kind}`}>{syncKindLabel(t, kind)}</Label>
                    <Switch
                        id={`${idPrefix}-${kind}`}
                        checked={values[kind]}
                        disabled={disabled}
                        onCheckedChange={(on) => onChange(kind, on)}
                    />
                </div>
            ))}
        </div>
    )
}

export function SyncMemberList({
    members,
    onRemove,
}: {
    members: SyncGroup['members']
    onRemove: (id: number) => void
}) {
    const { t } = useTranslation('settings')

    return (
        <ul className="divide-y rounded-lg border">
            {members.map((m) => (
                <li key={m.id} className="flex items-center justify-between gap-3 px-3 py-2 text-sm">
                    <span className="truncate">{m.name}</span>
                    <div className="flex items-center gap-2">
                        <Badge variant="secondary">{spaceRoleLabel(t, m.role)}</Badge>
                        {m.role === 'admin' && (
                            <AlertDialog>
                                <AlertDialogTrigger asChild>
                                    <Button variant="ghost" size="icon" title={t('spaces.sync.remove')}>
                                        <Trash2 className="size-4" />
                                    </Button>
                                </AlertDialogTrigger>
                                <AlertDialogContent>
                                    <AlertDialogHeader>
                                        <AlertDialogTitle>
                                            {t('spaces.sync.removeTitle', { name: m.name })}
                                        </AlertDialogTitle>
                                        <AlertDialogDescription>
                                            {t('spaces.sync.removeDescription')}
                                        </AlertDialogDescription>
                                    </AlertDialogHeader>
                                    <AlertDialogFooter>
                                        <AlertDialogCancel>{t('actions.cancel', { ns: 'common' })}</AlertDialogCancel>
                                        <AlertDialogAction onClick={() => onRemove(m.id)}>
                                            {t('spaces.sync.remove')}
                                        </AlertDialogAction>
                                    </AlertDialogFooter>
                                </AlertDialogContent>
                            </AlertDialog>
                        )}
                    </div>
                </li>
            ))}
        </ul>
    )
}

interface SyncSetupFormProps {
    space: Space
    candidates: Space[]
    onSetup: (change: SyncChange) => void
}

/** Starts a group for a space that has none: the spaces to join it with, and what to keep identical. */
export function SyncSetupForm({ space, candidates, onSetup }: SyncSetupFormProps) {
    const { t } = useTranslation('settings')
    const [picked, setPicked] = useState<number[]>([])
    const [setup, setSetup] = useState<Record<SyncKind, boolean>>({ categories: true, tags: true })

    return (
        <>
            <p className="text-sm text-muted-foreground">{t('spaces.sync.none')}</p>
            <div className="space-y-2">
                {candidates.map((c) => (
                    <div key={c.id} className="flex items-center gap-2 text-sm">
                        <Checkbox
                            id={`sync-space-${c.id}`}
                            checked={picked.includes(c.id)}
                            onCheckedChange={(on) =>
                                setPicked(on ? [...picked, c.id] : picked.filter((id) => id !== c.id))
                            }
                        />
                        <Label htmlFor={`sync-space-${c.id}`}>{c.name}</Label>
                    </div>
                ))}
            </div>
            <SyncKindSwitches
                idPrefix="setup"
                values={setup}
                onChange={(kind, on) => setSetup({ ...setup, [kind]: on })}
            />
            <Button
                disabled={!picked.length || !(setup.categories || setup.tags)}
                onClick={() => onSetup({ add_space_ids: [space.id, ...picked], ...setup })}
            >
                {t('spaces.sync.setup')}
            </Button>
        </>
    )
}
