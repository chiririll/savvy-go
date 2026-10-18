import type { ImportPreviewResult } from '@/types/import'

type PreviewSummary = ImportPreviewResult['summary']

/**
 * A big file is previewed from a sample. Then the number of transactions to create is
 * estimated by the row count of the whole file and shown with a tilde.
 */
export function previewEstimate(summary: PreviewSummary): { isSampled: boolean; createEstimate: number } {
    const isSampled = summary.totalRows !== null && summary.sampled < summary.totalRows
    return { isSampled, createEstimate: isSampled ? (summary.totalRows as number) : summary.willCreate }
}
