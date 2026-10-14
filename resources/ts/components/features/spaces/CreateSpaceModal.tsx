import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { ResponsiveDialog } from '@/components/shared/ResponsiveDialog'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { spacesApi } from '@/api/spaces'
import { getApiErrorMessage } from '@/lib/api-error'
import { useServerCurrencyCatalog, useSpaceContent, useSpacePresets, useSpaces, useSwitchSpace } from '@/hooks/use-spaces'
import { loadSpaceDraft, useSpaceDraftStore } from '@/stores/space-draft'
import type { Space } from '@/types/spaces'
import { InviteForm } from './InviteForm'
import { EntitiesStep } from './create/EntitiesStep'
import {
    buildContent,
    COPY_KINDS,
    draftProblems,
    nextStep,
    previousStep,
    toCopySource,
    toRequest,
    type CopyKind,
    type DraftContent,
    type DraftStep,
} from './create/draft'

interface Props {
    open: boolean
    onOpenChange: (open: boolean) => void
    /** A user without any space cannot close the modal. */
    dismissible?: boolean
}

/**
 * Builds a filled space in steps and sends it in one request. The draft lives
 * in this browser only (stores/space-draft.ts) and survives closing the modal.
 */
export function CreateSpaceModal({ open, onOpenChange, dismissible = true }: Props) {
    const { t } = useTranslation('settings')
    const draft = useSpaceDraftStore((state) => state.draft)
    const update = useSpaceDraftStore((state) => state.update)
    const clear = useSpaceDraftStore((state) => state.clear)
    const switchSpace = useSwitchSpace()
    const { data: spaces = [] } = useSpaces()
    const presets = useSpacePresets(open)
    const catalog = useServerCurrencyCatalog(open)
    const [loaded, setLoaded] = useState(false)
    const [created, setCreated] = useState<Space | null>(null)
    const [inviting, setInviting] = useState(false)

    const otherSpaces = spaces.length
    // Copying needs to read the other space; a source that is gone is ignored.
    const sourceId = draft.sourceId !== null && spaces.some((s) => s.id === draft.sourceId) ? draft.sourceId : null
    const source = useSpaceContent(open && draft.step === 'copy' ? sourceId : null)

    useEffect(() => {
        if (!open) return
        let cancelled = false
        setLoaded(false)
        void loadSpaceDraft().then(() => !cancelled && setLoaded(true))
        return () => {
            cancelled = true
        }
    }, [open])

    // A draft saved on the copy step can come back when there is nothing to copy from.
    const step: DraftStep = draft.step === 'copy' && otherSpaces === 0 ? 'name' : draft.step
    const problems = useMemo(() => draftProblems(draft), [draft])

    const create = useMutation({
        mutationFn: () => spacesApi.create(toRequest(draft)),
        onSuccess: (space) => {
            clear()
            setCreated(space)
        },
        onError: (error) => toast.error(getApiErrorMessage(error, t('spaces.create.failed'))),
    })

    const finish = async () => {
        const space = created
        setCreated(null)
        setInviting(false)
        onOpenChange(false)
        if (space) await switchSpace(space.id)
    }

    const requestClose = (next: boolean) => {
        if (next) return onOpenChange(true)
        if (created) return void finish()
        if (dismissible) onOpenChange(false)
    }

    const content: DraftContent = draft
    const kindLabels: Record<CopyKind, string> = {
        currencies: t('spaces.create.copy.kinds.currencies'),
        accounts: t('spaces.create.copy.kinds.accounts'),
        categories: t('spaces.create.copy.kinds.categories'),
        tags: t('spaces.create.copy.kinds.tags'),
    }
    const goTo = (to: DraftStep, patch: Partial<DraftContent> = {}) => update({ step: to, ...patch })
    const seed = () => {
        const target = nextStep(step, otherSpaces)
        const fromCopy = step === 'copy'
        const sourceContent = fromCopy && source.data ? toCopySource(source.data) : null
        // Leaving the name step for the entities does not discard edits made earlier.
        const seeded = draft.categories.length > 0 && !fromCopy
        goTo(target, seeded ? {} : buildContent(draft.copy, sourceContent, presets.data!))
    }

    const waiting = !loaded || !presets.data || !catalog.data

    let body: React.ReactNode
    let footer: React.ReactNode
    const back = step !== 'intro' && (
        <Button type="button" variant="outline" onClick={() => goTo(previousStep(step, otherSpaces))}>
            {t('actions.back', { ns: 'common' })}
        </Button>
    )
    const cancel = dismissible && (
        <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
            {t('actions.cancel', { ns: 'common' })}
        </Button>
    )

    if (created) {
        body = (
            <div className="space-y-4">
                <p className="text-sm">{t('spaces.create.done.text', { name: created.name })}</p>
                {inviting && <InviteForm spaceId={created.id} />}
            </div>
        )
        footer = (
            <>
                {!inviting && (
                    <Button type="button" variant="outline" onClick={() => setInviting(true)}>
                        {t('spaces.create.done.invite')}
                    </Button>
                )}
                <Button type="button" onClick={() => void finish()}>
                    {t('actions.done', { ns: 'common' })}
                </Button>
            </>
        )
    } else if (waiting) {
        body = <Loader2 className="mx-auto size-6 animate-spin text-muted-foreground" />
        footer = cancel
    } else if (step === 'intro') {
        body = (
            <ul className="list-disc space-y-2 pl-5 text-sm">
                <li>{t('spaces.create.intro.finances')}</li>
                <li>{t('spaces.create.intro.currency')}</li>
                <li>{t('spaces.create.intro.members')}</li>
                <li>{t('spaces.create.intro.separate')}</li>
            </ul>
        )
        footer = (
            <>
                {cancel}
                <Button type="button" onClick={() => goTo('name')}>{t('actions.continue', { ns: 'common' })}</Button>
            </>
        )
    } else if (step === 'name') {
        body = (
            <div className="space-y-2">
                <Label htmlFor="space-name">{t('spaces.fields.name')}</Label>
                <Input
                    id="space-name"
                    value={draft.name}
                    maxLength={100}
                    autoFocus
                    onChange={(e) => update({ name: e.target.value })}
                    onKeyDown={(e) => e.key === 'Enter' && draft.name.trim() && seed()}
                />
            </div>
        )
        footer = (
            <>
                {back}
                <Button type="button" disabled={!draft.name.trim()} onClick={seed}>
                    {t('actions.continue', { ns: 'common' })}
                </Button>
            </>
        )
    } else if (step === 'copy') {
        body = (
            <div className="space-y-4">
                <div className="space-y-1.5">
                    <Label>{t('spaces.create.copy.source')}</Label>
                    <Select
                        value={sourceId === null ? 'defaults' : String(sourceId)}
                        onValueChange={(v) => update({ sourceId: v === 'defaults' ? null : Number(v) })}
                    >
                        <SelectTrigger className="w-full">
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="defaults">{t('spaces.create.copy.defaults')}</SelectItem>
                            {spaces.map((space) => (
                                <SelectItem key={space.id} value={String(space.id)}>{space.name}</SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </div>
                {sourceId !== null && (
                    <fieldset className="space-y-2">
                        <legend className="text-sm font-medium">{t('spaces.create.copy.what')}</legend>
                        {COPY_KINDS.map((kind) => (
                            <label key={kind} className="flex items-center gap-2 text-sm">
                                <Checkbox
                                    checked={draft.copy[kind]}
                                    onCheckedChange={(checked) => update({ copy: { ...draft.copy, [kind]: checked === true } })}
                                />
                                {kindLabels[kind]}
                            </label>
                        ))}
                        <p className="text-xs text-muted-foreground">{t('spaces.create.copy.hint')}</p>
                    </fieldset>
                )}
            </div>
        )
        footer = (
            <>
                {back}
                <Button type="button" disabled={sourceId !== null && source.isPending} onClick={seed}>
                    {t('actions.continue', { ns: 'common' })}
                </Button>
            </>
        )
    } else if (step === 'entities') {
        body = <EntitiesStep content={content} catalog={catalog.data} onChange={(patch) => update(patch)} />
        footer = (
            <>
                {back}
                <Button type="button" disabled={!draft.baseCurrency || problems.includes('accounts') || problems.includes('balances')} onClick={() => goTo('confirm')}>
                    {t('actions.continue', { ns: 'common' })}
                </Button>
            </>
        )
    } else {
        const request = toRequest(draft)
        body = (
            <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
                <dt className="text-muted-foreground">{t('spaces.fields.name')}</dt>
                <dd>{request.name}</dd>
                <dt className="text-muted-foreground">{t('spaces.create.summary.base')}</dt>
                <dd>{request.baseCurrency}</dd>
                <dt className="text-muted-foreground">{t('spaces.create.summary.currencies')}</dt>
                <dd>{request.currencies?.length ? request.currencies.join(', ') : '—'}</dd>
                <dt className="text-muted-foreground">{t('spaces.create.summary.accounts')}</dt>
                <dd>{request.accounts?.length ?? 0}</dd>
                <dt className="text-muted-foreground">{t('spaces.create.summary.categories')}</dt>
                <dd>{request.categories?.length ?? 0}</dd>
                <dt className="text-muted-foreground">{t('spaces.create.summary.tags')}</dt>
                <dd>{request.tags?.length ?? 0}</dd>
            </dl>
        )
        footer = (
            <>
                {back}
                <Button type="button" disabled={problems.length > 0 || create.isPending} onClick={() => create.mutate()}>
                    {create.isPending && <Loader2 className="size-4 animate-spin" />}
                    {t('spaces.create.submit')}
                </Button>
            </>
        )
    }

    return (
        <ResponsiveDialog
            open={open}
            onOpenChange={requestClose}
            title={created ? t('spaces.create.done.title') : t('spaces.create.title')}
            description={created ? undefined : t('spaces.create.description')}
            className="sm:max-w-2xl"
            footer={footer}
        >
            {body}
        </ResponsiveDialog>
    )
}
