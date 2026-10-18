import { useTranslation } from 'react-i18next'
import { ArrowLeft, ArrowRight, Loader2, Upload } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { intlLocale } from '@/lib/i18n'
import type { ImportPreviewResult } from '@/types/import'
import { previewEstimate } from './preview-estimate'
import type { CsvImportFlow } from './useCsvImportFlow'

function importCountLabel(t: ReturnType<typeof useTranslation>['t'], previewResult: ImportPreviewResult | null) {
    if (!previewResult) return t('import.importCount', { tilde: '', count: 0 })
    const { isSampled, createEstimate } = previewEstimate(previewResult.summary)
    return t('import.importCount', {
        tilde: isSampled ? '~' : '',
        count: createEstimate.toLocaleString(intlLocale()),
    })
}

function Busy({ label }: { label: string }) {
    return (
        <>
            <Loader2 className="size-4 mr-2 animate-spin" />
            {label}
        </>
    )
}

/** How far the server got through the rows while importing. */
export function ImportProgress({ flow }: { flow: CsvImportFlow }) {
    const { t } = useTranslation('settings')
    const { importMutation } = flow

    if (flow.step !== 'preview' || !importMutation.isPending || !importMutation.progress) return null
    const { processed, total } = importMutation.progress

    return (
        <div className="space-y-2">
            <div className="flex justify-between text-sm text-muted-foreground">
                <span>{t('import.importingProgress')}</span>
                <span>
                    {processed}
                    {total ? ` / ${total}` : ''}
                </span>
            </div>
            <Progress value={total ? Math.round((processed / total) * 100) : 0} />
        </div>
    )
}

function PrimaryAction({ flow }: { flow: CsvImportFlow }) {
    const { t } = useTranslation('settings')
    const { step, isLoading } = flow

    if (step === 'upload') {
        return (
            <Button onClick={flow.next} disabled={!flow.parseResult || isLoading}>
                {t('import.next')}
                <ArrowRight className="size-4 ml-2" />
            </Button>
        )
    }
    if (step === 'mapping') {
        return (
            <Button type="submit" form="mapping-form" disabled={isLoading}>
                {flow.isPreviewing ? (
                    <Busy label={t('import.loading')} />
                ) : (
                    <>
                        {t('import.preview')}
                        <ArrowRight className="size-4 ml-2" />
                    </>
                )}
            </Button>
        )
    }
    return (
        <Button onClick={flow.runImport} disabled={isLoading || (flow.previewResult?.summary.willCreate ?? 0) === 0}>
            {flow.importMutation.isPending ? (
                <Busy label={t('import.importing')} />
            ) : (
                <>
                    <Upload className="size-4 mr-2" />
                    {importCountLabel(t, flow.previewResult)}
                </>
            )}
        </Button>
    )
}

export function ImportFooter({ flow }: { flow: CsvImportFlow }) {
    const { t } = useTranslation('settings')
    const { t: tCommon } = useTranslation('common')

    if (flow.step === 'result') {
        return (
            <div className="flex justify-center pt-6 border-t">
                <Button variant="outline" onClick={flow.reset}>
                    {t('import.importAnother')}
                </Button>
            </div>
        )
    }

    return (
        <div className="flex justify-between pt-6 border-t">
            <div>
                {flow.step !== 'upload' && (
                    <Button variant="outline" onClick={flow.back} disabled={flow.isLoading}>
                        <ArrowLeft className="size-4 mr-2" />
                        {tCommon('actions.back')}
                    </Button>
                )}
            </div>
            <div className="flex gap-2">
                <PrimaryAction flow={flow} />
            </div>
        </div>
    )
}
