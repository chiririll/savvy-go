import { cn } from '@/lib/utils'
import type { ImportCategoryMap, ImportPreviewResult } from '@/types/import'
import { CategoryMapping } from './CategoryMapping'
import { NewEntities, PreviewIssues, PreviewSummary, PreviewTable } from './PreviewParts'

interface PreviewStepProps {
    previewResult: ImportPreviewResult
    categoryMap: ImportCategoryMap
    onCategoryMapChange: (value: ImportCategoryMap) => void
    isLoading: boolean
}

export function PreviewStep({ previewResult, categoryMap, onCategoryMapChange, isLoading }: PreviewStepProps) {
    const { previewTransactions, summary } = previewResult

    return (
        <div className={cn('space-y-6', isLoading && 'pointer-events-none')} aria-busy={isLoading}>
            <PreviewSummary summary={summary} />

            <CategoryMapping
                candidates={summary.categories}
                value={categoryMap}
                onChange={onCategoryMapChange}
                disabled={isLoading}
            />

            <NewEntities summary={summary} />
            <PreviewTable transactions={previewTransactions} />
            <PreviewIssues transactions={previewTransactions} />
        </div>
    )
}
