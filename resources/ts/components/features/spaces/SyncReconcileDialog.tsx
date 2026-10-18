import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog'
import { useApplySync, useCategories, useSyncDiff } from '@/hooks'
import { SyncChange, SyncDiff } from '@/types/spaces'
import { SyncDiffCards } from './SyncDiffCards'
import {
    buildDecisions,
    buildDefaultDecisions,
    emptyChoices,
    isComplete,
    isIdentical,
    type ReconcileChoices,
} from './sync-reconcile'

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
    const [choices, setChoices] = useState<ReconcileChoices>(emptyChoices)
    const choose = (patch: { [K in keyof ReconcileChoices]?: ReconcileChoices[K] }) =>
        setChoices((current) => ({ ...current, ...patch }))

    const { mutate: loadDiff, reset: resetDiff } = diff
    useEffect(() => {
        setChoices(emptyChoices())
        resetDiff()
        if (change) loadDiff(change)
    }, [change, loadDiff, resetDiff])

    const data: SyncDiff | undefined = diff.data
    const spaceName = (id: number) => data?.spaces.find((s) => s.id === id)?.name ?? String(id)
    const names = (ids: number[]) => ids.map(spaceName).join(', ')

    const submit = () => {
        if (!change || !data) return
        apply.mutate(
            { ...change, decisions: buildDecisions(data, choices), defaults: buildDefaultDecisions(data, choices) },
            { onSuccess: onClose },
        )
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
                {isIdentical(data) && <p className="text-sm">{t('spaces.sync.dialog.identical')}</p>}

                {data && (
                    <SyncDiffCards
                        data={data}
                        choices={choices}
                        onChoose={choose}
                        mine={mine}
                        spaceName={spaceName}
                        names={names}
                    />
                )}

                <DialogFooter>
                    <Button variant="outline" onClick={onClose}>
                        {t('actions.cancel', { ns: 'common' })}
                    </Button>
                    <Button disabled={!data || !isComplete(data, choices) || apply.isPending} onClick={submit}>
                        {t('spaces.sync.dialog.apply')}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    )
}
