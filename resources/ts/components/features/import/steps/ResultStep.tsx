import { CheckCircle2, XCircle, Tag, FolderOpen, Coins } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { useNavigate } from 'react-router-dom'
import type { ImportResult } from '@/types/import'

interface ResultStepProps {
    result: ImportResult
}

export function ResultStep({ result }: ResultStepProps) {
    const { t } = useTranslation('settings')
    const navigate = useNavigate()

    const hasCreated = result.created > 0
    const hasErrors = result.errors.length > 0

    return (
        <div className="space-y-6">
            <div className={`p-6 border rounded-lg text-center ${hasCreated ? 'bg-success/10' : 'bg-warning/10'}`}>
                {hasCreated ? (
                    <CheckCircle2 className="size-16 text-success mx-auto mb-4" />
                ) : (
                    <XCircle className="size-16 text-warning mx-auto mb-4" />
                )}
                <h2 className="text-2xl font-bold mb-2">
                    {hasCreated ? t('import.complete') : t('import.noneImported')}
                </h2>
                <p className="text-muted-foreground">
                    {hasCreated
                        ? t('import.successCount', { count: result.created })
                        : t('import.allSkipped')}
                </p>
            </div>

            <div className="grid gap-4 sm:grid-cols-3">
                <div className="p-4 border rounded-lg text-center">
                    <div className="text-3xl font-bold text-success">{result.created}</div>
                    <div className="text-sm text-muted-foreground">{t('import.created')}</div>
                </div>
                <div className="p-4 border rounded-lg text-center">
                    <div className="text-3xl font-bold text-warning">{result.skippedDuplicates}</div>
                    <div className="text-sm text-muted-foreground">{t('import.skippedDuplicates')}</div>
                </div>
                <div className="p-4 border rounded-lg text-center">
                    <div className="text-3xl font-bold text-destructive">{result.errors.length}</div>
                    <div className="text-sm text-muted-foreground">{t('import.errors')}</div>
                </div>
            </div>

            {(result.createdCategories.length > 0 ||
                result.createdTags.length > 0 ||
                result.createdCurrencies.length > 0) && (
                <div className="p-4 border rounded-lg">
                    <h3 className="font-medium mb-3">{t('import.newEntitiesCreated')}</h3>
                    <div className="space-y-3">
                        {result.createdCategories.length > 0 && (
                            <div className="flex items-start gap-2">
                                <FolderOpen className="size-4 text-muted-foreground mt-0.5" />
                                <div>
                                    <div className="text-sm font-medium">{t('import.categoriesLabel')}</div>
                                    <div className="flex flex-wrap gap-1 mt-1">
                                        {result.createdCategories.map((cat) => (
                                            <Badge key={cat} variant="outline">{cat}</Badge>
                                        ))}
                                    </div>
                                </div>
                            </div>
                        )}
                        {result.createdTags.length > 0 && (
                            <div className="flex items-start gap-2">
                                <Tag className="size-4 text-muted-foreground mt-0.5" />
                                <div>
                                    <div className="text-sm font-medium">{t('import.tagsLabel')}</div>
                                    <div className="flex flex-wrap gap-1 mt-1">
                                        {result.createdTags.map((tag) => (
                                            <Badge key={tag} variant="outline">{tag}</Badge>
                                        ))}
                                    </div>
                                </div>
                            </div>
                        )}
                        {result.createdCurrencies.length > 0 && (
                            <div className="flex items-start gap-2">
                                <Coins className="size-4 text-muted-foreground mt-0.5" />
                                <div>
                                    <div className="text-sm font-medium">{t('import.currenciesLabel')}</div>
                                    <div className="flex flex-wrap gap-1 mt-1">
                                        {result.createdCurrencies.map((code) => (
                                            <Badge key={code} variant="outline">{code}</Badge>
                                        ))}
                                    </div>
                                </div>
                            </div>
                        )}
                    </div>
                </div>
            )}

            {hasErrors && (
                <div className="p-4 border rounded-lg bg-destructive/10">
                    <div className="flex items-center gap-2 mb-2">
                        <XCircle className="size-4 text-destructive" />
                        <span className="font-medium">{t('import.errorsCount', { count: result.errors.length })}</span>
                    </div>
                    {/* Scrolls when long, so it takes focus for keyboard scrolling. */}
                    <div className="max-h-[200px] overflow-y-auto" tabIndex={0}>
                        <ul className="text-sm space-y-1">
                            {result.errors.map((err, i) => (
                                <li key={i}>
                                    {t('import.rowDetail', { row: err.row, detail: err.message })}
                                </li>
                            ))}
                        </ul>
                    </div>
                </div>
            )}

            <div className="flex justify-center gap-4">
                <Button variant="outline" onClick={() => navigate('/transactions')}>
                    {t('import.viewTransactions')}
                </Button>
            </div>
        </div>
    )
}
