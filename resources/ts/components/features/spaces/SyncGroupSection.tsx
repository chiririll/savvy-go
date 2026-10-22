import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Waypoints } from 'lucide-react'
import { SettingsSection } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useApplySync, useSpaces, useSyncGroup } from '@/hooks'
import { testId } from '@/lib/test-id'
import { Space, SyncChange, SyncKind } from '@/types/spaces'
import { SyncReconcileDialog } from './SyncReconcileDialog'
import { SyncKindSwitches, SyncMemberList, SyncSetupForm } from './SyncGroupParts'

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
                        <SyncMemberList members={group.members} onRemove={remove} />
                        {group.hidden > 0 && (
                            <p className="text-xs text-muted-foreground">
                                {t('spaces.sync.hidden', { count: group.hidden })}
                            </p>
                        )}
                        <SyncKindSwitches
                            idPrefix="sync"
                            values={flags}
                            disabled={!isAdmin || apply.isPending}
                            onChange={switchKind}
                        />
                        <p className="text-xs text-muted-foreground">{t('spaces.sync.rights')}</p>
                        {isAdmin && candidates.length > 0 && (
                            <div className="flex flex-wrap gap-2">
                                <Select value={adding} onValueChange={setAdding}>
                                    <SelectTrigger className="flex-1" aria-label={t('spaces.sync.choose')}>
                                        <SelectValue placeholder={t('spaces.sync.choose')} />
                                    </SelectTrigger>
                                    <SelectContent>
                                        {candidates.map((c) => (
                                            <SelectItem key={c.id} value={String(c.id)}>
                                                {c.name}
                                            </SelectItem>
                                        ))}
                                    </SelectContent>
                                </Select>
                                <Button
                                    {...testId('sync-add')}
                                    disabled={!adding}
                                    onClick={() =>
                                        setChange({ group_id: group.id, add_space_ids: [Number(adding)], ...flags })
                                    }
                                >
                                    {t('spaces.sync.add')}
                                </Button>
                            </div>
                        )}
                    </>
                ) : candidates.length ? (
                    <SyncSetupForm space={space} candidates={candidates} onSetup={setChange} />
                ) : (
                    <p className="text-xs text-muted-foreground">{t('spaces.sync.howTo')}</p>
                )}
            </div>
            <SyncReconcileDialog change={change} onClose={() => setChange(null)} />
        </SettingsSection>
    )
}
