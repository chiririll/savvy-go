import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { ResponsiveDialog } from '@/components/shared/dialogs/ResponsiveDialog'
import { useApplySync, useCategories, useSyncDiff } from '@/hooks'
import { testId } from '@/lib/test-id'
import { SyncChange, SyncDiff } from '@/types/spaces'
import { SyncStepBody } from './SyncSteps'
import {
    buildDecisions,
    buildDefaultDecisions,
    emptyChoices,
    isComplete,
    isIdentical,
    syncSteps,
    type ReconcileChoices,
} from './sync-reconcile'

interface Props {
    /** The change to reconcile for; the dialog is open while it is set. */
    change: SyncChange | null
    onClose: () => void
}

/**
 * Shows how the spaces of a sync group differ and lets the caller decide, one step at a time:
 * categories, where the transactions of a deleted one go (only when needed), tags, then the
 * defaults and a summary. Nothing happens until Apply; every choice starts on the harmless side
 * (keep, add where missing), and Back keeps what was chosen.
 */
export function SyncReconcileDialog({ change, onClose }: Props) {
    const { t } = useTranslation(['settings', 'common'])
    const diff = useSyncDiff()
    const apply = useApplySync()
    const { data: mine = [] } = useCategories()
    const [choices, setChoices] = useState<ReconcileChoices>(emptyChoices)
    const [index, setIndex] = useState(0)
    const choose = (patch: { [K in keyof ReconcileChoices]?: ReconcileChoices[K] }) =>
        setChoices((current) => ({ ...current, ...patch }))

    const { mutate: loadDiff, reset: resetDiff } = diff
    useEffect(() => {
        setChoices(emptyChoices())
        setIndex(0)
        resetDiff()
        if (change) loadDiff(change)
    }, [change, loadDiff, resetDiff])

    const data: SyncDiff | undefined = diff.data
    const spaceName = (id: number) => data?.spaces.find((s) => s.id === id)?.name ?? String(id)
    const names = (ids: number[]) => ids.map(spaceName).join(', ')

    const steps = syncSteps(data, choices)
    const current = Math.min(index, steps.length - 1)
    const step = steps[current]
    const last = current === steps.length - 1
    const complete = !!data && isComplete(data, choices)

    const submit = () => {
        if (!change || !data) return
        apply.mutate(
            { ...change, decisions: buildDecisions(data, choices), defaults: buildDefaultDecisions(data, choices) },
            { onSuccess: onClose },
        )
    }

    const footer = (
        <>
            {current > 0 ? (
                <Button variant="outline" onClick={() => setIndex(current - 1)} {...testId('sync-back')}>
                    {t('spaces.sync.dialog.back')}
                </Button>
            ) : (
                <Button variant="outline" onClick={onClose}>
                    {t('actions.cancel', { ns: 'common' })}
                </Button>
            )}
            {data && !last && (
                <Button
                    disabled={step === 'merge' && !complete}
                    onClick={() => setIndex(current + 1)}
                    {...testId('sync-next')}
                >
                    {t('spaces.sync.dialog.next')}
                </Button>
            )}
            {data && last && (
                <Button disabled={!complete || apply.isPending} onClick={submit} {...testId('sync-apply')}>
                    {t('spaces.sync.dialog.apply')}
                </Button>
            )}
        </>
    )

    return (
        <ResponsiveDialog
            open={!!change}
            onOpenChange={(open) => !open && onClose()}
            title={t('spaces.sync.dialog.title')}
            description={
                data && steps.length > 1
                    ? t('spaces.sync.dialog.stepOf', {
                          step: t(`spaces.sync.dialog.steps.${step}`),
                          current: current + 1,
                          total: steps.length,
                      })
                    : t('spaces.sync.dialog.description')
            }
            className="sm:max-w-2xl"
            footer={footer}
        >
            {diff.isPending && <p className="text-sm text-muted-foreground">{t('spaces.sync.dialog.loading')}</p>}
            {diff.isError && <p className="text-sm text-destructive">{diff.error.message}</p>}
            {isIdentical(data) && <p className="text-sm">{t('spaces.sync.dialog.identical')}</p>}
            {data && !isIdentical(data) && (
                <div data-step={step} {...testId(`sync-step-${step}`)}>
                    <SyncStepBody
                        step={step}
                        data={data}
                        choices={choices}
                        onChoose={choose}
                        mine={mine}
                        spaceName={spaceName}
                        names={names}
                    />
                </div>
            )}
        </ResponsiveDialog>
    )
}
