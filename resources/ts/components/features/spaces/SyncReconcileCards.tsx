import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { syncTypeLabel } from '@/lib/labels'
import { testId } from '@/lib/test-id'
import { localizeDefaultName } from '@/lib/localized-name'
import type { SyncCategoryConflict, SyncTagConflict } from '@/types/spaces'
import { differs, defaultOptions, type SyncDefaultConflict } from './sync-reconcile'

const TRIGGER = 'h-8 w-full sm:w-72'

interface SpaceNames {
    spaceName: (id: number) => string
    names: (ids: number[]) => string
}

function Card({ id, children }: { id: string; children: ReactNode }) {
    return (
        <div className="space-y-2 rounded-lg border p-3 text-sm" {...testId(id)}>
            {children}
        </div>
    )
}

function KeepSelect({
    label,
    kept,
    keepLabel,
    dropLabel,
    onChange,
}: {
    /** What is kept or dropped, for a screen reader. */
    label: string
    kept: boolean
    keepLabel: string
    dropLabel: string
    onChange: (kept: boolean) => void
}) {
    return (
        <Select value={kept ? 'keep' : 'drop'} onValueChange={(v) => onChange(v === 'keep')}>
            <SelectTrigger className={TRIGGER} aria-label={label}>
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
    onKeep: (kept: boolean) => void
    onLook: (spaceId: number) => void
}

export function SyncCategoryCard({
    conflict: c,
    kept,
    look,
    spaceName,
    names,
    onKeep,
    onLook,
}: CategoryCardProps) {
    const { t } = useTranslation(['settings', 'common'])

    return (
        <Card id="sync-category">
            <div>
                <p className="font-medium">
                    {localizeDefaultName(c.name)} · {syncTypeLabel(t, c.type)}
                </p>
                <p className="text-xs text-muted-foreground">
                    {t('spaces.sync.dialog.presentIn', { spaces: names(c.present.map((p) => p.spaceId)) })}
                    {c.missing.length > 0 && ` · ${t('spaces.sync.dialog.missingIn', { spaces: names(c.missing) })}`}
                </p>
            </div>
            <KeepSelect
                label={localizeDefaultName(c.name)}
                kept={kept}
                keepLabel={c.missing.length ? t('spaces.sync.dialog.keepAdd') : t('spaces.sync.dialog.keep')}
                dropLabel={t('spaces.sync.dialog.drop')}
                onChange={onKeep}
            />
            {kept && differs(c) && (
                <div className="space-y-1">
                    <Label>{t('spaces.sync.dialog.look')}</Label>
                    <Select value={String(look ?? c.present[0].spaceId)} onValueChange={(v) => onLook(Number(v))}>
                        <SelectTrigger className={TRIGGER} aria-label={t('spaces.sync.dialog.look')}>
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
        <Card id="sync-default">
            <p className="font-medium">{t('spaces.sync.dialog.defaultTitle', { type: syncTypeLabel(t, d.type) })}</p>
            <p className="text-xs text-muted-foreground">
                {d.defaults.map((x) => `${spaceName(x.spaceId)}: ${localizeDefaultName(x.name) || '—'}`).join(' · ')}
            </p>
            <Select value={value} onValueChange={onChange}>
                <SelectTrigger
                    className={TRIGGER}
                    aria-label={t('spaces.sync.dialog.defaultTitle', { type: syncTypeLabel(t, d.type) })}
                >
                    <SelectValue />
                </SelectTrigger>
                <SelectContent>
                    {defaultOptions(d).map((name) => (
                        <SelectItem key={name} value={name}>
                            {localizeDefaultName(name)}
                        </SelectItem>
                    ))}
                </SelectContent>
            </Select>
        </Card>
    )
}

interface SuccessorCardProps {
    conflict: SyncCategoryConflict
    value: string | undefined
    options: string[]
    onChange: (name: string) => void
}

/** Where the transactions of a category that is deleted everywhere go. */
export function SyncSuccessorCard({ conflict: c, value, options, onChange }: SuccessorCardProps) {
    const { t } = useTranslation(['settings', 'common'])

    return (
        <Card id="sync-successor">
            <p className="font-medium">
                {localizeDefaultName(c.name)} · {syncTypeLabel(t, c.type)}
            </p>
            <div className="space-y-1">
                <Label>{t('spaces.sync.dialog.successor')}</Label>
                <Select value={value ?? ''} onValueChange={onChange}>
                    <SelectTrigger className={TRIGGER} aria-label={t('spaces.sync.dialog.successor')}>
                        <SelectValue placeholder={t('spaces.sync.dialog.successorPlaceholder')} />
                    </SelectTrigger>
                    <SelectContent>
                        {options.map((name) => (
                            <SelectItem key={name} value={name}>
                                {localizeDefaultName(name)}
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>
            </div>
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
        <Card id="sync-tag">
            <div>
                <p className="font-medium">#{localizeDefaultName(c.name, 'tags')}</p>
                <p className="text-xs text-muted-foreground">
                    {t('spaces.sync.dialog.presentIn', { spaces: names(c.present.map((p) => p.spaceId)) })}
                    {` · ${t('spaces.sync.dialog.missingIn', { spaces: names(c.missing) })}`}
                </p>
            </div>
            <KeepSelect
                label={`#${localizeDefaultName(c.name, 'tags')}`}
                kept={kept}
                keepLabel={t('spaces.sync.dialog.keepAdd')}
                dropLabel={t('spaces.sync.dialog.dropTag')}
                onChange={onKeep}
            />
        </Card>
    )
}
