import type { ReactNode } from 'react'
import { AlertTriangle, CheckCircle2, Copy, XCircle } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/badge'
import { cn, formatCurrency } from '@/lib/utils'
import { intlLocale } from '@/lib/i18n'
import { transactionTypeLabelLoose } from '@/lib/labels'
import type { ImportPreviewResult } from '@/types/import'
import { previewEstimate } from '../preview-estimate'

type Summary = ImportPreviewResult['summary']
type PreviewTransaction = ImportPreviewResult['previewTransactions'][number]

const nf = (n: number) => n.toLocaleString(intlLocale())

function Tile({ tone, value, label }: { tone: 'green' | 'yellow' | 'red'; value: string; label: string }) {
    const classes = {
        green: ['bg-success/10', 'text-success'],
        yellow: ['bg-warning/10', 'text-warning'],
        red: ['bg-destructive/10', 'text-destructive'],
    }[tone]

    return (
        <div className={cn('p-4 border rounded-lg', classes[0])}>
            <div className={cn('text-3xl font-bold', classes[1])}>{value}</div>
            <div className="text-sm text-muted-foreground">{label}</div>
        </div>
    )
}

export function PreviewSummary({ summary }: { summary: Summary }) {
    const { t } = useTranslation('settings')
    const { isSampled, createEstimate } = previewEstimate(summary)

    return (
        <>
            <div className="grid gap-4 sm:grid-cols-3">
                <Tile
                    tone="green"
                    value={`${isSampled ? '~' : ''}${nf(createEstimate)}`}
                    label={t('import.willCreate')}
                />
                <Tile
                    tone="yellow"
                    value={nf(summary.willSkip)}
                    label={isSampled ? t('import.duplicatesSkippedSample') : t('import.duplicatesSkipped')}
                />
                <Tile
                    tone="red"
                    value={nf(summary.hasErrors)}
                    label={isSampled ? t('import.errorsSample') : t('import.errors')}
                />
            </div>

            {isSampled && (
                <p className="text-sm text-muted-foreground">
                    {t('import.sampledNote', { sampled: nf(summary.sampled), total: nf(summary.totalRows as number) })}
                </p>
            )}
        </>
    )
}

function BadgeRow({ label, items }: { label: string; items: string[] }) {
    if (items.length === 0) return null

    return (
        <div className="flex flex-wrap gap-1">
            <span className="text-muted-foreground">{label}</span>
            {items.map((item) => (
                <Badge key={item} variant="outline">
                    {item}
                </Badge>
            ))}
        </div>
    )
}

/** What the import adds besides the transactions. */
export function NewEntities({ summary }: { summary: Summary }) {
    const { t } = useTranslation('settings')
    // With the category mapping panel, new categories are managed there instead.
    const newCategories = summary.categories.length > 0 ? [] : summary.categoriesToCreate

    if (newCategories.length === 0 && summary.tagsToCreate.length === 0 && summary.currenciesToCreate.length === 0) {
        return null
    }

    return (
        <div className="p-4 border rounded-lg bg-info/10">
            <div className="flex items-center gap-2 mb-2">
                <AlertTriangle className="size-4 text-info" />
                <span className="font-medium">{t('import.newEntities')}</span>
            </div>
            <div className="space-y-2 text-sm">
                <BadgeRow label={t('import.categories')} items={newCategories} />
                <BadgeRow label={t('import.tags')} items={summary.tagsToCreate} />
                <BadgeRow label={t('import.currencies')} items={summary.currenciesToCreate} />
            </div>
        </div>
    )
}

export function PreviewTable({ transactions }: { transactions: PreviewTransaction[] }) {
    const { t } = useTranslation('settings')
    const { t: tPages } = useTranslation('pages')

    const statusConfig = {
        new: { icon: CheckCircle2, color: 'text-success', bgColor: 'bg-success/10', label: t('import.statusNew') },
        duplicate: {
            icon: Copy,
            color: 'text-warning',
            bgColor: 'bg-warning/10',
            label: t('import.statusDuplicate'),
        },
        error: { icon: XCircle, color: 'text-destructive', bgColor: 'bg-destructive/10', label: t('import.statusError') },
    }

    return (
        <div className="border rounded-lg overflow-hidden">
            <div tabIndex={0} className="overflow-x-auto max-h-[400px] overflow-y-auto">
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
                        {transactions.map((tx) => {
                            const config = statusConfig[tx.status]
                            const StatusIcon = config.icon

                            return (
                                <tr key={tx.row} className={cn('border-t', config.bgColor)}>
                                    <td className="px-4 py-2 text-muted-foreground">{tx.row}</td>
                                    <td className="px-4 py-2">
                                        <div className="flex items-center gap-1">
                                            <StatusIcon className={cn('size-4', config.color)} />
                                            <span className={config.color}>{config.label}</span>
                                        </div>
                                    </td>
                                    <td className="px-4 py-2 font-mono">{tx.date || '-'}</td>
                                    <td className="px-4 py-2">
                                        <Badge variant={tx.type === 'income' ? 'default' : 'secondary'}>
                                            {tx.type ? transactionTypeLabelLoose(tPages, tx.type) : '-'}
                                        </Badge>
                                    </td>
                                    <td className="px-4 py-2 text-right font-mono">
                                        {tx.amount != null ? formatCurrency(tx.amount) : '-'}
                                    </td>
                                    <td className="px-4 py-2 truncate max-w-[200px]">{tx.description || '-'}</td>
                                    <td className="px-4 py-2 truncate max-w-[150px]">{tx.category || '-'}</td>
                                </tr>
                            )
                        })}
                    </tbody>
                </table>
            </div>
        </div>
    )
}

const MAX_ISSUES = 10

interface IssueListProps {
    tone: 'yellow' | 'red'
    icon: ReactNode
    title: string
    /** Row and detail of each transaction that has something to report. */
    issues: { row: number; detail: string }[]
}

function IssueList({ tone, icon, title, issues }: IssueListProps) {
    const { t } = useTranslation('settings')
    if (issues.length === 0) return null

    return (
        <div className={cn('p-4 border rounded-lg', tone === 'yellow' ? 'bg-warning/10' : 'bg-destructive/10')}>
            <div className="flex items-center gap-2 mb-2">
                {icon}
                <span className="font-medium">{title}</span>
            </div>
            <ul className="text-sm space-y-1">
                {issues.slice(0, MAX_ISSUES).map(({ row, detail }) => (
                    <li key={row}>{t('import.rowDetail', { row, detail })}</li>
                ))}
            </ul>
        </div>
    )
}

export function PreviewIssues({ transactions }: { transactions: PreviewTransaction[] }) {
    const { t } = useTranslation('settings')

    return (
        <>
            <IssueList
                tone="yellow"
                icon={<AlertTriangle className="size-4 text-warning" />}
                title={t('import.warnings')}
                issues={transactions
                    .filter((tx) => tx.warnings.length > 0)
                    .map((tx) => ({ row: tx.row, detail: tx.warnings.join(', ') }))}
            />
            <IssueList
                tone="red"
                icon={<XCircle className="size-4 text-destructive" />}
                title={t('import.errorsSkipped')}
                issues={transactions
                    .filter((tx) => tx.error)
                    .map((tx) => ({ row: tx.row, detail: tx.error as string }))}
            />
        </>
    )
}
