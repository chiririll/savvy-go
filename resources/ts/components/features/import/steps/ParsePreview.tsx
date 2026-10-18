import { useTranslation } from 'react-i18next'
import type { CsvParseResult } from '@/types/import'

const PREVIEW_ROWS = 5

function Format({ label, value }: { label: string; value: string }) {
    return (
        <div className="flex items-center gap-2">
            <span className="text-muted-foreground">{label}</span>
            <span className="font-mono bg-muted px-2 py-0.5 rounded">{value}</span>
        </div>
    )
}

/** What was read from the uploaded file: the detected formats and its first rows. */
export function ParsePreview({ parseResult }: { parseResult: CsvParseResult }) {
    const { t } = useTranslation('settings')
    const { detectedFormats } = parseResult

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <h3 className="font-medium">{t('import.filePreview')}</h3>
                <span className="text-sm text-muted-foreground">
                    {t('import.rowsDetected', { count: parseResult.totalRows })}
                </span>
            </div>

            <div className="flex gap-4 text-sm">
                <Format label={t('import.dateFormat')} value={detectedFormats.dateFormat} />
                <Format label={t('import.amountFormat')} value={detectedFormats.amountFormat} />
                <Format
                    label={t('import.delimiter')}
                    value={detectedFormats.delimiter === '\t' ? 'TAB' : detectedFormats.delimiter}
                />
            </div>

            <div className="border rounded-lg overflow-hidden">
                <div className="overflow-x-auto">
                    <table className="w-full text-sm">
                        <thead className="bg-muted">
                            <tr>
                                {parseResult.headers.map((header, i) => (
                                    <th key={i} className="px-4 py-2 text-left font-medium">
                                        {header}
                                    </th>
                                ))}
                            </tr>
                        </thead>
                        <tbody>
                            {parseResult.previewRows.slice(0, PREVIEW_ROWS).map((row, i) => (
                                <tr key={i} className="border-t">
                                    {row.map((cell, j) => (
                                        <td key={j} className="px-4 py-2 truncate max-w-[200px]">
                                            {cell}
                                        </td>
                                    ))}
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
                {parseResult.previewRows.length > PREVIEW_ROWS && (
                    <div className="px-4 py-2 text-sm text-muted-foreground bg-muted/50 border-t">
                        {t('import.moreRows', { count: parseResult.totalRows - PREVIEW_ROWS })}
                    </div>
                )}
            </div>
        </div>
    )
}
