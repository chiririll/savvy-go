import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Trash2, Waypoints } from 'lucide-react'
import { SettingsSection } from '@/components/shared'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
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
import { useApplySync, useSpaces, useSyncGroup } from '@/hooks'
import { spaceRoleLabel, syncKindLabel } from '@/lib/labels'
import { Space, SyncChange, SyncKind } from '@/types/spaces'
import { SyncReconcileDialog } from './SyncReconcileDialog'

const KINDS: SyncKind[] = ['categories', 'tags']

/**
 * The sync group of a space: spaces that keep their categories and tags
 * identical. Joining a group or switching a kind on first opens the
 * reconciliation dialog; switching off and leaving need none.
 */
export function SyncGroupSection({ space, isAdmin }: { space: Space; isAdmin: boolean }) {
    const { t } = useTranslation('settings')
    const { data: group } = useSyncGroup(space.id)
    const { data: mine = [] } = useSpaces()
    const apply = useApplySync()
    const [change, setChange] = useState<SyncChange | null>(null)
    const [picked, setPicked] = useState<number[]>([])
    const [setup, setSetup] = useState<Record<SyncKind, boolean>>({ categories: true, tags: true })
    const [adding, setAdding] = useState('')

    // A group is built by an admin of every space in it.
    const inGroup = new Set([space.id, ...(group?.members.map((m) => m.id) ?? [])])
    const candidates = mine.filter((s) => s.role === 'admin' && !inGroup.has(s.id))

    if (!group && !isAdmin) return null

    const flags = { categories: !!group?.categories, tags: !!group?.tags }

    const switchKind = (kind: SyncKind, on: boolean) => {
        if (!group) return
        const next = { ...flags, [kind]: on }
        if (on) {
            setChange({ group_id: group.id, ...next })
        } else {
            apply.mutate({ group_id: group.id, ...next, decisions: [], defaults: [] })
        }
    }

    const remove = (id: number) => {
        if (!group) return
        apply.mutate({ group_id: group.id, remove_space_ids: [id], ...flags, decisions: [], defaults: [] })
    }

    return (
        <SettingsSection icon={Waypoints} title={t('spaces.sync.title')} description={t('spaces.sync.description')}>
            <div className="space-y-4">
                {group ? (
                    <>
                        <ul className="divide-y rounded-lg border">
                            {group.members.map((m) => (
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
                                                        <AlertDialogTitle>{t('spaces.sync.removeTitle', { name: m.name })}</AlertDialogTitle>
                                                        <AlertDialogDescription>{t('spaces.sync.removeDescription')}</AlertDialogDescription>
                                                    </AlertDialogHeader>
                                                    <AlertDialogFooter>
                                                        <AlertDialogCancel>{t('actions.cancel', { ns: 'common' })}</AlertDialogCancel>
                                                        <AlertDialogAction onClick={() => remove(m.id)}>{t('spaces.sync.remove')}</AlertDialogAction>
                                                    </AlertDialogFooter>
                                                </AlertDialogContent>
                                            </AlertDialog>
                                        )}
                                    </div>
                                </li>
                            ))}
                        </ul>
                        {group.hidden > 0 && (
                            <p className="text-xs text-muted-foreground">{t('spaces.sync.hidden', { count: group.hidden })}</p>
                        )}
                        <div className="space-y-2">
                            {KINDS.map((kind) => (
                                <div key={kind} className="flex items-center justify-between gap-3 text-sm">
                                    <Label htmlFor={`sync-${kind}`}>{syncKindLabel(t, kind)}</Label>
                                    <Switch
                                        id={`sync-${kind}`}
                                        checked={flags[kind]}
                                        disabled={!isAdmin || apply.isPending}
                                        onCheckedChange={(on) => switchKind(kind, on)}
                                    />
                                </div>
                            ))}
                        </div>
                        <p className="text-xs text-muted-foreground">{t('spaces.sync.rights')}</p>
                        {isAdmin && (
                            <div className="flex flex-wrap gap-2">
                                {candidates.length > 0 && (
                                    <>
                                        <Select value={adding} onValueChange={setAdding}>
                                            <SelectTrigger className="flex-1">
                                                <SelectValue placeholder={t('spaces.sync.choose')} />
                                            </SelectTrigger>
                                            <SelectContent>
                                                {candidates.map((c) => (
                                                    <SelectItem key={c.id} value={String(c.id)}>{c.name}</SelectItem>
                                                ))}
                                            </SelectContent>
                                        </Select>
                                        <Button
                                            disabled={!adding}
                                            onClick={() => setChange({ group_id: group.id, add_space_ids: [Number(adding)], ...flags })}
                                        >
                                            {t('spaces.sync.add')}
                                        </Button>
                                    </>
                                )}
                            </div>
                        )}
                    </>
                ) : candidates.length ? (
                    <>
                        <p className="text-sm text-muted-foreground">{t('spaces.sync.none')}</p>
                        <div className="space-y-2">
                            {candidates.map((c) => (
                                <div key={c.id} className="flex items-center gap-2 text-sm">
                                    <Checkbox
                                        id={`sync-space-${c.id}`}
                                        checked={picked.includes(c.id)}
                                        onCheckedChange={(on) => setPicked(on ? [...picked, c.id] : picked.filter((id) => id !== c.id))}
                                    />
                                    <Label htmlFor={`sync-space-${c.id}`}>{c.name}</Label>
                                </div>
                            ))}
                        </div>
                        <div className="space-y-2">
                            {KINDS.map((kind) => (
                                <div key={kind} className="flex items-center justify-between gap-3 text-sm">
                                    <Label htmlFor={`setup-${kind}`}>{syncKindLabel(t, kind)}</Label>
                                    <Switch
                                        id={`setup-${kind}`}
                                        checked={setup[kind]}
                                        onCheckedChange={(on) => setSetup({ ...setup, [kind]: on })}
                                    />
                                </div>
                            ))}
                        </div>
                        <Button
                            disabled={!picked.length || !(setup.categories || setup.tags)}
                            onClick={() => setChange({ add_space_ids: [space.id, ...picked], ...setup })}
                        >
                            {t('spaces.sync.setup')}
                        </Button>
                    </>
                ) : (
                    <p className="text-xs text-muted-foreground">{t('spaces.sync.howTo')}</p>
                )}
            </div>
            <SyncReconcileDialog change={change} onClose={() => setChange(null)} />
        </SettingsSection>
    )
}
