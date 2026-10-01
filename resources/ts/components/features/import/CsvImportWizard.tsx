import { useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Progress } from '@/components/ui/progress'
import { Stepper, Step } from '@/components/ui/stepper'
import { ArrowLeft, ArrowRight, Upload, Loader2, AlertCircle } from 'lucide-react'
import { useMultipartUpload } from '@/hooks/use-upload'
import { useParseImport, usePreviewImport, useExecuteImport } from '@/hooks/use-csv-import'
import { UploadStep } from './steps/UploadStep'
import { MappingStep } from './steps/MappingStep'
import { PreviewStep } from './steps/PreviewStep'
import { ResultStep } from './steps/ResultStep'
import { intlLocale } from '@/lib/i18n'
import type {
    ImportStep,
    CsvParseResult,
    ColumnMapping,
    ImportOptions,
    ImportPreviewResult,
    ImportResult,
    ImportCategoryMap,
} from '@/types/import'

const STEP_IDS: ImportStep[] = ['upload', 'mapping', 'preview', 'result']

const UPLOAD_BUCKET = 'transaction-imports'

export function CsvImportWizard() {
    const { t } = useTranslation('settings')
    const { t: tCommon } = useTranslation('common')
    const [step, setStep] = useState<ImportStep>('upload')
    const [parseResult, setParseResult] = useState<CsvParseResult | null>(null)
    const [previewResult, setPreviewResult] = useState<ImportPreviewResult | null>(null)
    const [importResult, setImportResult] = useState<ImportResult | null>(null)
    const [mapping, setMapping] = useState<ColumnMapping | null>(null)
    const [options, setOptions] = useState<ImportOptions | null>(null)
    const [categoryMap, setCategoryMap] = useState<ImportCategoryMap>({})
    const [error, setError] = useState<string | null>(null)

    const upload = useMultipartUpload()
    const parseMutation = useParseImport()
    const previewMutation = usePreviewImport()
    const importMutation = useExecuteImport()

    const currentStepIndex = STEP_IDS.findIndex((s) => s === step)

    const handleFileSelect = useCallback(async (file: File) => {
        setError(null)
        try {
            const { uploadId } = await upload.upload({ bucket: UPLOAD_BUCKET, file })
            const result = await parseMutation.mutateAsync(uploadId)
            setParseResult(result)
        } catch (e) {
            if (e instanceof DOMException && e.name === 'AbortError') {
                return
            }
            setError(e instanceof Error ? e.message : t('import.processFailed'))
        }
    }, [upload, parseMutation, t])

    const handleMappingSubmit = useCallback(async (newMapping: ColumnMapping, newOptions: ImportOptions) => {
        if (!parseResult) return

        setMapping(newMapping)
        setOptions(newOptions)
        setError(null)

        try {
            const result = await previewMutation.mutateAsync({
                importId: parseResult.importId,
                mapping: newMapping,
                options: newOptions,
            })
            setPreviewResult(result)
            // Existing categories are linked by default, missing ones are created.
            setCategoryMap(Object.fromEntries(
                result.summary.categories.map((c) => [c.name, c.matchId ?? 'create'])
            ))
            setStep('preview')
        } catch (e) {
            setError(e instanceof Error ? e.message : tCommon('toasts.import.previewFailed'))
        }
    }, [parseResult, previewMutation, tCommon])

    const handleImport = useCallback(async () => {
        if (!parseResult || !mapping || !options) return

        setError(null)

        try {
            const result = await importMutation.mutateAsync({
                importId: parseResult.importId,
                mapping,
                options: { ...options, categoryMap },
            })
            setImportResult(result)
            setStep('result')
        } catch (e) {
            setError(e instanceof Error ? e.message : tCommon('toasts.import.executeFailed'))
        }
    }, [parseResult, mapping, options, categoryMap, importMutation, tCommon])

    const handleBack = useCallback(() => {
        if (step === 'mapping') {
            setStep('upload')
        } else if (step === 'preview') {
            setStep('mapping')
        }
    }, [step])

    const handleNext = useCallback(() => {
        if (step === 'upload' && parseResult) {
            setStep('mapping')
        }
    }, [step, parseResult])

    const reset = useCallback(() => {
        setStep('upload')
        setParseResult(null)
        setPreviewResult(null)
        setImportResult(null)
        setMapping(null)
        setOptions(null)
        setCategoryMap({})
        setError(null)
        upload.reset()
    }, [upload])

    const isUploading = upload.isUploading
    const isParsing = parseMutation.isPending
    const isLoading = isUploading || isParsing || previewMutation.isPending || importMutation.isPending

    const canGoToStep = (targetIndex: number): boolean => {
        if (targetIndex === 0) return true
        if (targetIndex === 1) return !!parseResult
        if (targetIndex === 2) return !!previewResult
        if (targetIndex === 3) return !!importResult
        return false
    }

    const handleStepChange = (targetIndex: number) => {
        if (canGoToStep(targetIndex) && !isLoading) {
            setStep(STEP_IDS[targetIndex])
        }
    }

    return (
        <div className="space-y-8">
            <Stepper
                activeStep={currentStepIndex}
                onStepChange={handleStepChange}
                className="w-full"
            >
                {STEP_IDS.map((id, index) => (
                    <Step key={id} disabled={!canGoToStep(index) || isLoading}>
                        {t(`import.steps.${id}`)}
                    </Step>
                ))}
            </Stepper>

            <div className="min-h-[400px]">
                {step === 'upload' && (
                    <UploadStep
                        onFileSelect={handleFileSelect}
                        onCancelUpload={upload.cancel}
                        parseResult={parseResult}
                        isUploading={isUploading}
                        isParsing={isParsing}
                        uploadPercentage={upload.progress?.percentage ?? 0}
                        error={error}
                    />
                )}

                {step === 'mapping' && parseResult && (
                    <MappingStep
                        parseResult={parseResult}
                        onSubmit={handleMappingSubmit}
                        isLoading={previewMutation.isPending}
                    />
                )}

                {step === 'preview' && previewResult && (
                    <PreviewStep
                        previewResult={previewResult}
                        categoryMap={categoryMap}
                        onCategoryMapChange={setCategoryMap}
                        isLoading={importMutation.isPending}
                    />
                )}

                {step === 'result' && importResult && (
                    <ResultStep result={importResult} />
                )}
            </div>

            {error && step !== 'upload' && (
                <Alert variant="destructive">
                    <AlertCircle className="size-4" />
                    <AlertDescription>{error}</AlertDescription>
                </Alert>
            )}

            {step === 'preview' && importMutation.isPending && importMutation.progress && (
                <div className="space-y-2">
                    <div className="flex justify-between text-sm text-muted-foreground">
                        <span>{t('import.importingProgress')}</span>
                        <span>
                            {importMutation.progress.processed}
                            {importMutation.progress.total ? ` / ${importMutation.progress.total}` : ''}
                        </span>
                    </div>
                    <Progress
                        value={
                            importMutation.progress.total
                                ? Math.round((importMutation.progress.processed / importMutation.progress.total) * 100)
                                : 0
                        }
                    />
                </div>
            )}

            {step !== 'result' && (
                <div className="flex justify-between pt-6 border-t">
                    <div>
                        {step !== 'upload' && (
                            <Button
                                variant="outline"
                                onClick={handleBack}
                                disabled={isLoading}
                            >
                                <ArrowLeft className="size-4 mr-2" />
                                {tCommon('actions.back')}
                            </Button>
                        )}
                    </div>
                    <div className="flex gap-2">
                        {step === 'upload' && (
                            <Button
                                onClick={handleNext}
                                disabled={!parseResult || isLoading}
                            >
                                {t('import.next')}
                                <ArrowRight className="size-4 ml-2" />
                            </Button>
                        )}
                        {step === 'mapping' && (
                            <Button
                                type="submit"
                                form="mapping-form"
                                disabled={isLoading}
                            >
                                {previewMutation.isPending ? (
                                    <>
                                        <Loader2 className="size-4 mr-2 animate-spin" />
                                        {t('import.loading')}
                                    </>
                                ) : (
                                    <>
                                        {t('import.preview')}
                                        <ArrowRight className="size-4 ml-2" />
                                    </>
                                )}
                            </Button>
                        )}
                        {step === 'preview' && (
                            <Button
                                onClick={handleImport}
                                disabled={isLoading || (previewResult?.summary.willCreate ?? 0) === 0}
                            >
                                {importMutation.isPending ? (
                                    <>
                                        <Loader2 className="size-4 mr-2 animate-spin" />
                                        {t('import.importing')}
                                    </>
                                ) : (
                                    <>
                                        <Upload className="size-4 mr-2" />
                                        {(() => {
                                            const s = previewResult?.summary
                                            if (!s) return t('import.importCount', { tilde: '', count: 0 })
                                            const sampled = s.totalRows !== null && s.sampled < s.totalRows
                                            return t('import.importCount', {
                                                tilde: sampled ? '~' : '',
                                                count: (sampled ? (s.totalRows as number) : s.willCreate).toLocaleString(intlLocale()),
                                            })
                                        })()}
                                    </>
                                )}
                            </Button>
                        )}
                    </div>
                </div>
            )}

            {step === 'result' && (
                <div className="flex justify-center pt-6 border-t">
                    <Button variant="outline" onClick={reset}>
                        {t('import.importAnother')}
                    </Button>
                </div>
            )}
        </div>
    )
}
