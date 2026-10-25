import { useTranslation } from 'react-i18next'
import { Loader2 } from 'lucide-react'
import { InfiniteScrollSentinel } from '@/components/shared'
import { Button } from '@/components/ui/button'

interface TransactionsListFooterProps {
    loadedCount: number
    total?: number
    lastPage?: number
    hasNextPage: boolean
    isFetchingNextPage: boolean
    isFetchNextPageError: boolean
    onLoadMore: () => void
}

/** Count, load-more state and the scroll sentinel under an infinite list. */
export function TransactionsListFooter({
    loadedCount,
    total,
    lastPage = 1,
    hasNextPage,
    isFetchingNextPage,
    isFetchNextPageError,
    onLoadMore,
}: TransactionsListFooterProps) {
    const { t } = useTranslation('pages')

    return (
        <div className="mt-4 min-w-0 space-y-3">
            {total !== undefined && (
                <p className="text-sm text-muted-foreground">
                    {t('common:table.showing', {
                        from: 1,
                        to: loadedCount,
                        total,
                        label: t('transactions.itemLabel'),
                    })}
                </p>
            )}
            {isFetchNextPageError && (
                <div className="flex flex-col items-start gap-2">
                    <p className="text-sm text-muted-foreground">{t('common:table.loadMoreError')}</p>
                    <Button variant="outline" size="sm" onClick={onLoadMore}>
                        {t('common:actions.retry')}
                    </Button>
                </div>
            )}
            {isFetchingNextPage && (
                <div className="flex items-center gap-2 text-sm text-muted-foreground" role="status" aria-live="polite">
                    <Loader2 className="size-4 animate-spin" />
                    {t('common:table.loadingMore')}
                </div>
            )}
            {!hasNextPage && !isFetchingNextPage && lastPage > 1 && (
                <p className="text-sm text-muted-foreground">{t('common:table.endOfList')}</p>
            )}
            <InfiniteScrollSentinel
                enabled={hasNextPage && !isFetchingNextPage && !isFetchNextPageError}
                onVisible={onLoadMore}
            />
        </div>
    )
}
