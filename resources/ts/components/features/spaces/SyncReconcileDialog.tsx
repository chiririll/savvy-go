import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog'
import { useApplySync, useCategories, useSyncDiff } from '@/hooks'
import { syncTypeLabel } from '@/lib/labels'
import {
    SyncCategoryConflict,
    SyncChange,
    SyncDecision,
    SyncDiff,
    SyncTagConflict,
} from '@/types/spaces'

const categoryKey = (c: Pick<SyncCategoryConflict, 'type' | 'name'>) => `c:${c.type}:${c.name.toLowerCase()}`
const tagKey = (c: Pick<SyncTagConflict, 'name'>) => `t:${c.name.toLowerCase()}`

const differs = (c: SyncCategoryConflict) =>
    c.present.some((p) => (p.icon ?? '') !== (c.present[0].icon ?? '') || (p.color ?? '') !== (c.present[0].color ?? ''))

interface Props {
    /** The change to reconcile for; the dialog is open while it is set. */
    change: SyncChange | null
    onClose: () => void
}

/**
 * Shows how the spaces of a sync group differ and lets the caller decide, for
 * each difference, what stays. Nothing happens until Apply; every choice starts
 * on the harmless side (keep, add where missing).
 */
export function SyncReconcileDialog({ change, onClose }: Props) {
    const { t } = useTranslation(['settings', 'common'])
    const diff = useSyncDiff()
    const apply = useApplySync()
    const { data: mine = [] } = useCategories()
    const [keep, setKeep] = useState<Record<string, boolean>>({})
    const [look, setLook] = useState<Record<string, number>>({})
    const [successor, setSuccessor] = useState<Record<string, string>>({})
    const [defaults, setDefaults] = useState<Record<string, string>>({})

    const { mutate: loadDiff, reset: resetDiff } = diff
    useEffect(() => {
        setKeep({})
        setLook({})
        setSuccessor({})
        setDefaults({})
        resetDiff()
        if (change) loadDiff(change)
    }, [change, loadDiff, resetDiff])

    const data: SyncDiff | undefined = diff.data
    const spaceName = (id: number) => data?.spaces.find((s) => s.id === id)?.name ?? String(id)
    const names = (ids: number[]) => ids.map(spaceName).join(', ')
    const kept = (key: string) => keep[key] ?? true

    // Categories every space has already: where the transactions of a dropped one can go.
    const conflictNames = new Set((data?.categories ?? []).map(categoryKey))
    const successors = (type: string) =>
        mine.filter((c) => c.type === type && !conflictNames.has(categoryKey(c))).map((c) => c.name)

    const needsSuccessor = (c: SyncCategoryConflict) => !kept(categoryKey(c)) && c.present.some((p) => p.transactions > 0)
    const defaultChoice = (type: string, names: string[]) => defaults[type] ?? names[0]

    const decisions = (): SyncDecision[] => [
        ...(data?.categories ?? []).map((c): SyncDecision => {
            const key = categoryKey(c)
            const k = kept(key)
            return {
                kind: 'categories',
                type: c.type,
                name: c.name,
                keep: k,
                from_space_id: k && differs(c) ? (look[key] ?? c.present[0].spaceId) : undefined,
                successor: !k ? successor[key] : undefined,
            }
        }),
        ...(data?.tags ?? []).map((c): SyncDecision => ({ kind: 'tags', name: c.name, keep: kept(tagKey(c)) })),
    ]

    const defaultDecisions = () =>
        (data?.defaults ?? []).map((d) => {
            const options = [...new Set(d.defaults.map((x) => x.name).filter(Boolean))]
            return { type: d.type, name: defaultChoice(d.type, options) }
        })

    const complete = !!data && data.categories.every((c) => !needsSuccessor(c) || !!successor[categoryKey(c)])
    const identical = !!data && !data.categories.length && !data.tags.length && !data.defaults.length

    const submit = () => {
        if (!change || !data) return
        apply.mutate({ ...change, decisions: decisions(), defaults: defaultDecisions() }, { onSuccess: onClose })
    }

    return (
        <Dialog open={!!change} onOpenChange={(open) => !open && onClose()}>
            <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
                <DialogHeader>
                    <DialogTitle>{t('spaces.sync.dialog.title')}</DialogTitle>
                    <DialogDescription>{t('spaces.sync.dialog.description')}</DialogDescription>
                </DialogHeader>

                {diff.isPending && <p className="text-sm text-muted-foreground">{t('spaces.sync.dialog.loading')}</p>}
                {diff.isError && <p className="text-sm text-destructive">{diff.error.message}</p>}
                {identical && <p className="text-sm">{t('spaces.sync.dialog.identical')}</p>}

                <div className="space-y-4">
                    {data?.categories.map((c) => {
                        const key = categoryKey(c)
                        return (
                            <div key={key} className="space-y-2 rounded-lg border p-3 text-sm" data-testid="sync-category">
                                <div>
                                    <p className="font-medium">
                                        {c.name} · {syncTypeLabel(t, c.type)}
                                    </p>
                                    <p className="text-xs text-muted-foreground">
                                        {t('spaces.sync.dialog.presentIn', { spaces: names(c.present.map((p) => p.spaceId)) })}
                                        {c.missing.length > 0 && ` · ${t('spaces.sync.dialog.missingIn', { spaces: names(c.missing) })}`}
                                    </p>
                                </div>
                                <Select value={kept(key) ? 'keep' : 'drop'} onValueChange={(v) => setKeep({ ...keep, [key]: v === 'keep' })}>
                                    <SelectTrigger className="h-8 w-full sm:w-72">
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectItem value="keep">
                                            {c.missing.length ? t('spaces.sync.dialog.keepAdd') : t('spaces.sync.dialog.keep')}
                                        </SelectItem>
                                        <SelectItem value="drop">{t('spaces.sync.dialog.drop')}</SelectItem>
                                    </SelectContent>
                                </Select>
                                {kept(key) && differs(c) && (
                                    <div className="space-y-1">
                                        <Label>{t('spaces.sync.dialog.look')}</Label>
                                        <Select
                                            value={String(look[key] ?? c.present[0].spaceId)}
                                            onValueChange={(v) => setLook({ ...look, [key]: Number(v) })}
                                        >
                                            <SelectTrigger className="h-8 w-full sm:w-72">
                                                <SelectValue />
                                            </SelectTrigger>
                                            <SelectContent>
                                                {c.present.map((p) => (
                                                    <SelectItem key={p.spaceId} value={String(p.spaceId)}>
                                                        {spaceName(p.spaceId)} {p.icon ?? ''} {p.color ?? ''}
                                                    </SelectItem>
                                                ))}
                                            </SelectContent>
                                        </Select>
                                    </div>
                                )}
                                {needsSuccessor(c) && (
                                    <div className="space-y-1">
                                        <Label>{t('spaces.sync.dialog.successor')}</Label>
                                        <Select value={successor[key] ?? ''} onValueChange={(v) => setSuccessor({ ...successor, [key]: v })}>
                                            <SelectTrigger className="h-8 w-full sm:w-72">
                                                <SelectValue placeholder={t('spaces.sync.dialog.successorPlaceholder')} />
                                            </SelectTrigger>
                                            <SelectContent>
                                                {successors(c.type).map((name) => (
                                                    <SelectItem key={name} value={name}>{name}</SelectItem>
                                                ))}
                                            </SelectContent>
                                        </Select>
                                    </div>
                                )}
                            </div>
                        )
                    })}

                    {data?.defaults.map((d) => {
                        const options = [...new Set(d.defaults.map((x) => x.name).filter(Boolean))]
                        return (
                            <div key={d.type} className="space-y-2 rounded-lg border p-3 text-sm" data-testid="sync-default">
                                <p className="font-medium">{t('spaces.sync.dialog.defaultTitle', { type: syncTypeLabel(t, d.type) })}</p>
                                <p className="text-xs text-muted-foreground">
                                    {d.defaults.map((x) => `${spaceName(x.spaceId)}: ${x.name || '—'}`).join(' · ')}
                                </p>
                                <Select value={defaultChoice(d.type, options)} onValueChange={(v) => setDefaults({ ...defaults, [d.type]: v })}>
                                    <SelectTrigger className="h-8 w-full sm:w-72">
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        {options.map((name) => (
                                            <SelectItem key={name} value={name}>{name}</SelectItem>
                                        ))}
                                    </SelectContent>
                                </Select>
                            </div>
                        )
                    })}

                    {data?.tags.map((c) => {
                        const key = tagKey(c)
                        return (
                            <div key={key} className="space-y-2 rounded-lg border p-3 text-sm" data-testid="sync-tag">
                                <div>
                                    <p className="font-medium">#{c.name}</p>
                                    <p className="text-xs text-muted-foreground">
                                        {t('spaces.sync.dialog.presentIn', { spaces: names(c.present.map((p) => p.spaceId)) })}
                                        {` · ${t('spaces.sync.dialog.missingIn', { spaces: names(c.missing) })}`}
                                    </p>
                                </div>
                                <Select value={kept(key) ? 'keep' : 'drop'} onValueChange={(v) => setKeep({ ...keep, [key]: v === 'keep' })}>
                                    <SelectTrigger className="h-8 w-full sm:w-72">
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectItem value="keep">{t('spaces.sync.dialog.keepAdd')}</SelectItem>
                                        <SelectItem value="drop">{t('spaces.sync.dialog.dropTag')}</SelectItem>
                                    </SelectContent>
                                </Select>
                            </div>
                        )
                    })}
                </div>

                <DialogFooter>
                    <Button variant="outline" onClick={onClose}>{t('actions.cancel', { ns: 'common' })}</Button>
                    <Button disabled={!data || !complete || apply.isPending} onClick={submit}>
                        {t('spaces.sync.dialog.apply')}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    )
}
