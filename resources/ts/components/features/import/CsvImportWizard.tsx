import { useTranslation } from 'react-i18next'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Stepper, Step } from '@/components/ui/stepper'
import { AlertCircle } from 'lucide-react'
import { UploadStep } from './steps/UploadStep'
import { MappingStep } from './steps/MappingStep'
import { PreviewStep } from './steps/PreviewStep'
import { ResultStep } from './steps/ResultStep'
import { ImportFooter, ImportProgress } from './ImportFooter'
import { STEP_IDS, useCsvImportFlow } from './useCsvImportFlow'
import { importStepLabel } from '@/lib/labels'

export function CsvImportWizard() {
    const { t } = useTranslation('settings')
    const flow = useCsvImportFlow()
    const { step, parseResult, previewResult, importResult, isLoading } = flow

    return (
        <div className="space-y-8">
            <Stepper activeStep={STEP_IDS.indexOf(step)} onStepChange={flow.goToStep} className="w-full">
                {STEP_IDS.map((id, index) => (
                    <Step key={id} disabled={!flow.canGoToStep(index) || isLoading}>
                        {importStepLabel(t, id)}
                    </Step>
                ))}
            </Stepper>

            <div className="min-h-[400px]">
                {step === 'upload' && (
                    <UploadStep
                        onFileSelect={flow.selectFile}
                        onCancelUpload={flow.upload.cancel}
                        parseResult={parseResult}
                        isUploading={flow.upload.isUploading}
                        isParsing={flow.isParsing}
                        uploadPercentage={flow.upload.progress?.percentage ?? 0}
                        error={flow.error}
                    />
                )}

                {step === 'mapping' && parseResult && (
                    <MappingStep
                        parseResult={parseResult}
                        onSubmit={flow.submitMapping}
                        isLoading={flow.isPreviewing}
                    />
                )}

                {step === 'preview' && previewResult && (
                    <PreviewStep
                        previewResult={previewResult}
                        categoryMap={flow.categoryMap}
                        onCategoryMapChange={flow.setCategoryMap}
                        isLoading={flow.importMutation.isPending}
                    />
                )}

                {step === 'result' && importResult && <ResultStep result={importResult} />}
            </div>

            {flow.error && step !== 'upload' && (
                <Alert variant="destructive">
                    <AlertCircle className="size-4" />
                    <AlertDescription>{flow.error}</AlertDescription>
                </Alert>
            )}

            <ImportProgress flow={flow} />
            <ImportFooter flow={flow} />
        </div>
    )
}
