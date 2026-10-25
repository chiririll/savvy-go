import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { getApiErrorMessage } from '@/lib/api-error'
import { useMultipartUpload } from '@/hooks/use-upload'
import { useExecuteImport, useParseImport, usePreviewImport } from '@/hooks/use-csv-import'
import type {
    ColumnMapping,
    CsvParseResult,
    ImportCategoryMap,
    ImportOptions,
    ImportPreviewResult,
    ImportResult,
    ImportStep,
} from '@/types/import'

export const STEP_IDS: ImportStep[] = ['upload', 'mapping', 'preview', 'result']

const UPLOAD_BUCKET = 'transaction-imports'

const errorMessage = getApiErrorMessage

/** The state of the import wizard and what moves it from one step to the next. */
export function useCsvImportFlow() {
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

    const isLoading =
        upload.isUploading || parseMutation.isPending || previewMutation.isPending || importMutation.isPending

    const selectFile = async (file: File) => {
        setError(null)
        try {
            const { uploadId } = await upload.upload({ bucket: UPLOAD_BUCKET, file })
            setParseResult(await parseMutation.mutateAsync(uploadId))
        } catch (e) {
            if (e instanceof DOMException && e.name === 'AbortError') return
            setError(errorMessage(e, t('import.processFailed')))
        }
    }

    const submitMapping = async (newMapping: ColumnMapping, newOptions: ImportOptions) => {
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
            setCategoryMap(Object.fromEntries(result.summary.categories.map((c) => [c.name, c.matchId ?? 'create'])))
            setStep('preview')
        } catch (e) {
            setError(errorMessage(e, tCommon('toasts.import.previewFailed')))
        }
    }

    const runImport = async () => {
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
            setError(errorMessage(e, tCommon('toasts.import.executeFailed')))
        }
    }

    const back = () => {
        if (step === 'mapping') setStep('upload')
        else if (step === 'preview') setStep('mapping')
    }

    const next = () => {
        if (step === 'upload' && parseResult) setStep('mapping')
    }

    const reset = () => {
        setStep('upload')
        setParseResult(null)
        setPreviewResult(null)
        setImportResult(null)
        setMapping(null)
        setOptions(null)
        setCategoryMap({})
        setError(null)
        upload.reset()
    }

    // A finished step can be revisited; a later one only once its result exists.
    const reached = [true, !!parseResult, !!previewResult, !!importResult]
    const canGoToStep = (index: number) => reached[index] ?? false

    const goToStep = (index: number) => {
        if (canGoToStep(index) && !isLoading) setStep(STEP_IDS[index])
    }

    return {
        step,
        parseResult,
        previewResult,
        importResult,
        categoryMap,
        setCategoryMap,
        error,
        upload,
        isParsing: parseMutation.isPending,
        isPreviewing: previewMutation.isPending,
        importMutation,
        isLoading,
        selectFile,
        submitMapping,
        runImport,
        back,
        next,
        reset,
        canGoToStep,
        goToStep,
    }
}

export type CsvImportFlow = ReturnType<typeof useCsvImportFlow>
