import { CheckCircle2, XCircle, AlertTriangle, Copy } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/badge'
import { cn, formatCurrency } from '@/lib/utils'
import { intlLocale } from '@/lib/i18n'
import type { ImportCategoryMap, ImportPreviewResult } from '@/types/import'
import { CategoryMapping } from './CategoryMapping'

interface PreviewStepProps {
    previewResult: ImportPreviewResult
    categoryMap: ImportCategoryMap
    onCategoryMapChange: (value: ImportCategoryMap) => void
    isLoading: boolean
}

export function PreviewStep({ previewResult, categoryMap, onCategoryMapChange, isLoading }: PreviewStepProps) {
    const { t } = useTranslation('settings')
    const { t: tPages } = useTranslation('pages')
    const { previewTransactions, summary } = previewResult

    const isSampled = summary.totalRows !== null && summary.sampled < summary.totalRows
    const createEstimate = isSampled ? (summary.totalRows as number) : summary.willCreate
    const nf = (n: number) => n.toLocaleString(intlLocale())
    // With the category mapping panel, new categories are managed there instead.
    const newCategories = summary.categories.length > 0 ? [] : summary.categoriesToCreate

    const statusConfig = {
        new: {
            icon: CheckCircle2,
            color: 'text-green-500',
            bgColor: 'bg-green-500/10',
            label: t('import.statusNew'),
        },
        duplicate: {
            icon: Copy,
            color: 'text-yellow-500',
            bgColor: 'bg-yellow-500/10',
            label: t('import.statusDuplicate'),
        },
        error: {
            icon: XCircle,
            color: 'text-red-500',
            bgColor: 'bg-red-500/10',
            label: t('import.statusError'),
        },
    }

    return (
        <div className={cn('space-y-6', isLoading && 'opacity-50 pointer-events-none')}>
            <div className="grid gap-4 sm:grid-cols-3">
                <div className="p-4 border rounded-lg bg-green-500/10">
                    <div className="text-3xl font-bold text-green-600">
                        {isSampled && '~'}{nf(createEstimate)}
                    </div>
                    <div className="text-sm text-muted-foreground">{t('import.willCreate')}</div>
                </div>
                <div className="p-4 border rounded-lg bg-yellow-500/10">
                    <div className="text-3xl font-bold text-yellow-600">{nf(summary.willSkip)}</div>
                    <div className="text-sm text-muted-foreground">
                        {isSampled ? t('import.duplicatesSkippedSample') : t('import.duplicatesSkipped')}
                    </div>
                </div>
                <div className="p-4 border rounded-lg bg-red-500/10">
                    <div className="text-3xl font-bold text-red-600">{nf(summary.hasErrors)}</div>
                    <div className="text-sm text-muted-foreground">
                        {isSampled ? t('import.errorsSample') : t('import.errors')}
                    </div>
                </div>
            </div>

            {isSampled && (
                <p className="text-sm text-muted-foreground">
                    {t('import.sampledNote', { sampled: nf(summary.sampled), total: nf(summary.totalRows as number) })}
                </p>
            )}

            <CategoryMapping
                candidates={summary.categories}
                value={categoryMap}
                onChange={onCategoryMapChange}
                disabled={isLoading}
            />

            {(newCategories.length > 0 ||
                summary.tagsToCreate.length > 0 ||
                summary.currenciesToCreate.length > 0) && (
                <div className="p-4 border rounded-lg bg-blue-500/10">
                    <div className="flex items-center gap-2 mb-2">
                        <AlertTriangle className="size-4 text-blue-500" />
                        <span className="font-medium">{t('import.newEntities')}</span>
                    </div>
                    <div className="space-y-2 text-sm">
                        {newCategories.length > 0 && (
                            <div className="flex flex-wrap gap-1">
                                <span className="text-muted-foreground">{t('import.categories')}</span>
                                {newCategories.map((cat) => (
                                    <Badge key={cat} variant="outline">{cat}</Badge>
                                ))}
                            </div>
                        )}
                        {summary.tagsToCreate.length > 0 && (
                            <div className="flex flex-wrap gap-1">
                                <span className="text-muted-foreground">{t('import.tags')}</span>
                                {summary.tagsToCreate.map((tag) => (
                                    <Badge key={tag} variant="outline">{tag}</Badge>
                                ))}
                            </div>
                        )}
                        {summary.currenciesToCreate.length > 0 && (
                            <div className="flex flex-wrap gap-1">
                                <span className="text-muted-foreground">{t('import.currencies')}</span>
                                {summary.currenciesToCreate.map((cur) => (
                                    <Badge key={cur} variant="outline">{cur}</Badge>
                                ))}
                            </div>
                        )}
                    </div>
                </div>
            )}

            <div className="border rounded-lg overflow-hidden">
                <div className="overflow-x-auto max-h-[400px] overflow-y-auto">
                    <table className="w-full text-sm">
                        <thead className="bg-muted sticky top-0">
                            <tr>
                                <th className="px-4 py-2 text-left font-medium w-16">{t('import.colRow')}</th>
                                <th className="px-4 py-2 text-left font-medium w-24">{t('import.colStatus')}</th>
                                <th className="px-4 py-2 text-left font-medium w-28">{t('import.colDate')}</th>
                                <th className="px-4 py-2 text-left font-medium w-20">{t('import.colType')}</th>
                                <th className="px-4 py-2 text-right font-medium w-28">{t('import.colAmount')}</th>
                                <th className="px-4 py-2 text-left font-medium">{t('import.colDescription')}</th>
                                <th className="px-4 py-2 text-left font-medium w-32">{t('import.colCategory')}</th>
                            </tr>
                        </thead>
                        <tbody>
                            {previewTransactions.map((tx) => {
                                const config = statusConfig[tx.status]
                                const StatusIcon = config.icon

                                return (
                                    <tr key={tx.row} className={cn('border-t', config.bgColor)}>
                                        <td className="px-4 py-2 text-muted-foreground">
                                            {tx.row}
                                        </td>
                                        <td className="px-4 py-2">
                                            <div className="flex items-center gap-1">
                                                <StatusIcon className={cn('size-4', config.color)} />
                                                <span className={config.color}>{config.label}</span>
                                            </div>
                                        </td>
                                        <td className="px-4 py-2 font-mono">
                                            {tx.date || '-'}
                                        </td>
                                        <td className="px-4 py-2">
                                            <Badge variant={tx.type === 'income' ? 'default' : 'secondary'}>
                                                {tx.type
                                                    ? tPages(`transactions.types.${tx.type}`, { defaultValue: tx.type })
                                                    : '-'}
                                            </Badge>
                                        </td>
                                        <td className="px-4 py-2 text-right font-mono">
                                            {tx.amount != null ? formatCurrency(tx.amount) : '-'}
                                        </td>
                                        <td className="px-4 py-2 truncate max-w-[200px]">
                                            {tx.description || '-'}
                                        </td>
                                        <td className="px-4 py-2 truncate max-w-[150px]">
                                            {tx.category || '-'}
                                        </td>
                                    </tr>
                                )
                            })}
                        </tbody>
                    </table>
                </div>
            </div>

            {previewTransactions.some((tx) => tx.warnings.length > 0) && (
                <div className="p-4 border rounded-lg bg-yellow-500/10">
                    <div className="flex items-center gap-2 mb-2">
                        <AlertTriangle className="size-4 text-yellow-500" />
                        <span className="font-medium">{t('import.warnings')}</span>
                    </div>
                    <ul className="text-sm space-y-1">
                        {previewTransactions
                            .filter((tx) => tx.warnings.length > 0)
                            .slice(0, 10)
                            .map((tx) => (
                                <li key={tx.row}>
                                    {t('import.rowDetail', { row: tx.row, detail: tx.warnings.join(', ') })}
                                </li>
                            ))}
                    </ul>
                </div>
            )}

            {previewTransactions.some((tx) => tx.error) && (
                <div className="p-4 border rounded-lg bg-red-500/10">
                    <div className="flex items-center gap-2 mb-2">
                        <XCircle className="size-4 text-red-500" />
                        <span className="font-medium">{t('import.errorsSkipped')}</span>
                    </div>
                    <ul className="text-sm space-y-1">
                        {previewTransactions
                            .filter((tx) => tx.error)
                            .slice(0, 10)
                            .map((tx) => (
                                <li key={tx.row}>
                                    {t('import.rowDetail', { row: tx.row, detail: tx.error })}
                                </li>
                            ))}
                    </ul>
                </div>
            )}
        </div>
    )
}
