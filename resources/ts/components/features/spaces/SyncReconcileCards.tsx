import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { syncTypeLabel } from '@/lib/labels'
import type { SyncCategoryConflict, SyncTagConflict } from '@/types/spaces'
import { differs, defaultOptions, type SyncDefaultConflict } from './sync-reconcile'

const TRIGGER = 'h-8 w-full sm:w-72'

interface SpaceNames {
    spaceName: (id: number) => string
    names: (ids: number[]) => string
}

function Card({ testId, children }: { testId: string; children: ReactNode }) {
    return (
        <div className="space-y-2 rounded-lg border p-3 text-sm" data-testid={testId}>
            {children}
        </div>
    )
}

function KeepSelect({
    kept,
    keepLabel,
    dropLabel,
    onChange,
}: {
    kept: boolean
    keepLabel: string
    dropLabel: string
    onChange: (kept: boolean) => void
}) {
    return (
        <Select value={kept ? 'keep' : 'drop'} onValueChange={(v) => onChange(v === 'keep')}>
            <SelectTrigger className={TRIGGER}>
                <SelectValue />
            </SelectTrigger>
            <SelectContent>
                <SelectItem value="keep">{keepLabel}</SelectItem>
                <SelectItem value="drop">{dropLabel}</SelectItem>
            </SelectContent>
        </Select>
    )
}

interface CategoryCardProps extends SpaceNames {
    conflict: SyncCategoryConflict
    kept: boolean
    look: number | undefined
    successor: string | undefined
    needsSuccessor: boolean
    successors: string[]
    onKeep: (kept: boolean) => void
    onLook: (spaceId: number) => void
    onSuccessor: (name: string) => void
}

export function SyncCategoryCard({
    conflict: c,
    kept,
    look,
    successor,
    needsSuccessor,
    successors,
    spaceName,
    names,
    onKeep,
    onLook,
    onSuccessor,
}: CategoryCardProps) {
    const { t } = useTranslation(['settings', 'common'])

    return (
        <Card testId="sync-category">
            <div>
                <p className="font-medium">
                    {c.name} · {syncTypeLabel(t, c.type)}
                </p>
                <p className="text-xs text-muted-foreground">
                    {t('spaces.sync.dialog.presentIn', { spaces: names(c.present.map((p) => p.spaceId)) })}
                    {c.missing.length > 0 && ` · ${t('spaces.sync.dialog.missingIn', { spaces: names(c.missing) })}`}
                </p>
            </div>
            <KeepSelect
                kept={kept}
                keepLabel={c.missing.length ? t('spaces.sync.dialog.keepAdd') : t('spaces.sync.dialog.keep')}
                dropLabel={t('spaces.sync.dialog.drop')}
                onChange={onKeep}
            />
            {kept && differs(c) && (
                <div className="space-y-1">
                    <Label>{t('spaces.sync.dialog.look')}</Label>
                    <Select value={String(look ?? c.present[0].spaceId)} onValueChange={(v) => onLook(Number(v))}>
                        <SelectTrigger className={TRIGGER}>
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
            {needsSuccessor && (
                <div className="space-y-1">
                    <Label>{t('spaces.sync.dialog.successor')}</Label>
                    <Select value={successor ?? ''} onValueChange={onSuccessor}>
                        <SelectTrigger className={TRIGGER}>
                            <SelectValue placeholder={t('spaces.sync.dialog.successorPlaceholder')} />
                        </SelectTrigger>
                        <SelectContent>
                            {successors.map((name) => (
                                <SelectItem key={name} value={name}>
                                    {name}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </div>
            )}
        </Card>
    )
}

interface DefaultCardProps {
    conflict: SyncDefaultConflict
    value: string
    spaceName: (id: number) => string
    onChange: (name: string) => void
}

export function SyncDefaultCard({ conflict: d, value, spaceName, onChange }: DefaultCardProps) {
    const { t } = useTranslation(['settings', 'common'])

    return (
        <Card testId="sync-default">
            <p className="font-medium">{t('spaces.sync.dialog.defaultTitle', { type: syncTypeLabel(t, d.type) })}</p>
            <p className="text-xs text-muted-foreground">
                {d.defaults.map((x) => `${spaceName(x.spaceId)}: ${x.name || '—'}`).join(' · ')}
            </p>
            <Select value={value} onValueChange={onChange}>
                <SelectTrigger className={TRIGGER}>
                    <SelectValue />
                </SelectTrigger>
                <SelectContent>
                    {defaultOptions(d).map((name) => (
                        <SelectItem key={name} value={name}>
                            {name}
                        </SelectItem>
                    ))}
                </SelectContent>
            </Select>
        </Card>
    )
}

interface TagCardProps extends Pick<SpaceNames, 'names'> {
    conflict: SyncTagConflict
    kept: boolean
    onKeep: (kept: boolean) => void
}

export function SyncTagCard({ conflict: c, kept, names, onKeep }: TagCardProps) {
    const { t } = useTranslation(['settings', 'common'])

    return (
        <Card testId="sync-tag">
            <div>
                <p className="font-medium">#{c.name}</p>
                <p className="text-xs text-muted-foreground">
                    {t('spaces.sync.dialog.presentIn', { spaces: names(c.present.map((p) => p.spaceId)) })}
                    {` · ${t('spaces.sync.dialog.missingIn', { spaces: names(c.missing) })}`}
                </p>
            </div>
            <KeepSelect
                kept={kept}
                keepLabel={t('spaces.sync.dialog.keepAdd')}
                dropLabel={t('spaces.sync.dialog.dropTag')}
                onChange={onKeep}
            />
        </Card>
    )
}
