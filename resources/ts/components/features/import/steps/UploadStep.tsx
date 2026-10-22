import { useTranslation } from 'react-i18next'
import { AlertCircle, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import type { CsvParseResult } from '@/types/import'
import { DropZone } from './DropZone'
import { ParsePreview } from './ParsePreview'

interface UploadStepProps {
    onFileSelect: (file: File) => void
    onCancelUpload: () => void
    parseResult: CsvParseResult | null
    isUploading: boolean
    isParsing: boolean
    uploadPercentage: number
    error: string | null
}

export function UploadStep({
    onFileSelect,
    onCancelUpload,
    parseResult,
    isUploading,
    isParsing,
    uploadPercentage,
    error,
}: UploadStepProps) {
    const { t } = useTranslation('settings')
    const { t: tCommon } = useTranslation('common')
    const isLoading = isUploading || isParsing

    return (
        <div className="space-y-6">
            <DropZone onFileSelect={onFileSelect} disabled={isLoading} />

            {isUploading && (
                <div className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                        <span className="text-muted-foreground">{t('import.uploading')}</span>
                        <div className="flex items-center gap-3">
                            <span className="font-mono">{uploadPercentage}%</span>
                            <Button variant="ghost" size="sm" onClick={onCancelUpload}>
                                {tCommon('actions.cancel')}
                            </Button>
                        </div>
                    </div>
                    <Progress value={uploadPercentage} />
                </div>
            )}

            {isParsing && (
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <Loader2 className="size-4 animate-spin" />
                    <span>{t('import.parsing')}</span>
                </div>
            )}

            {error && (
                <div className="flex items-center gap-2 p-4 bg-destructive/10 text-destructive rounded-lg">
                    <AlertCircle className="size-5" />
                    <span>{error}</span>
                </div>
            )}

            {parseResult && !isLoading && <ParsePreview parseResult={parseResult} />}
        </div>
    )
}
