import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2, Plus } from 'lucide-react'
import { Page, PageHeader, InfiniteScrollSentinel, useNegativeBalanceConfirm } from '@/components/shared'
import { Button } from '@/components/ui/button'
import {
    ApplyDeferredDateDialog,
    TransactionFiltersPanel,
    TransactionList,
    UpcomingPendingStrip,
    useTransactionDeepLink,
} from '@/components/features/transactions'
import { Transaction } from '@/types'
import {
    useInfiniteTransactions,
    useTransactions,
    useDeleteTransaction,
    useDuplicateTransaction,
    useConfirmTransaction,
    useSkipTransaction,
    useCategories,
    useTags,
    useTransactionListFilters,
} from '@/hooks'
import { useReadOnly } from '@/components/providers/ReadOnlyProvider'
import { warningsForConfirmedDuplicate } from '@/lib/negative-balance'
import { addDaysLocal, cn } from '@/lib/utils'

export default function TransactionsPage() {
    const { t } = useTranslation('pages')
    const list = useTransactionListFilters()
    const {
        transactions,
        meta,
        isLoading,
        isError,
        isFetchingNextPage,
        isFetchNextPageError,
        hasNextPage,
        fetchNextPage,
        refetch,
    } = useInfiniteTransactions(list.filters)
    const { data: upcomingPending } = useTransactions({
        status: 'pending',
        end_date: addDaysLocal(new Date(), 7),
        sort_by: 'date',
        sort_direction: 'asc',
        per_page: 20,
    })
    const deleteTransaction = useDeleteTransaction()
    const duplicateTransaction = useDuplicateTransaction()
    const confirmTransaction = useConfirmTransaction()
    const skipTransaction = useSkipTransaction()
    const { data: categories } = useCategories()
    const { data: tags } = useTags()
    const isReadOnly = useReadOnly()
    const { openCreate, openEdit } = useTransactionDeepLink(transactions)
    const [applying, setApplying] = useState<Transaction | null>(null)
    const { confirmIfNeeded, dialog: negativeBalanceDialog } = useNegativeBalanceConfirm<number>()
    const loadedCount = transactions.length
    const total = meta?.total
    const showListError = isError && loadedCount === 0 && !isLoading

    const handleCreate = () => {
        openCreate(list.params.type ? { type: list.params.type } : undefined)
    }

    const highlight = upcomingPending?.data ?? []
    const showHighlightColumn = highlight.length > 0
    const showHighlightItems = showHighlightColumn && !list.params.status

    return (
        <Page title={t('transactions.title')}>
            <PageHeader
                title={t('transactions.title')}
                description={t('transactions.description')}
                onCreateClick={isReadOnly ? undefined : handleCreate}
                createLabel={t('transactions.create')}
            />

            <div
                className={cn(
                    'mx-auto grid w-full max-w-[800px] gap-4',
                    showHighlightColumn && 'lg:max-w-[1144px] lg:grid-cols-[minmax(0,800px)_320px] lg:justify-center',
                )}
            >
            {showHighlightColumn && (
                <div className="min-w-0 lg:col-start-2 lg:row-start-1">
                    {showHighlightItems && (
                        <UpcomingPendingStrip
                            transactions={highlight}
                            isReadOnly={isReadOnly}
                            onConfirm={setApplying}
                            onSkip={(id) => skipTransaction.mutate(id)}
                        />
                    )}
                </div>
            )}

            <div className="min-w-0 lg:col-start-1 lg:row-start-1">
                <TransactionFiltersPanel
                    list={list}
                    categories={categories}
                    tags={tags}
                />

                {showListError ? (
                    <div className="rounded-lg border py-16 text-center">
                        <p className="text-sm text-muted-foreground">{t('common:errors.somethingWrong')}</p>
                        <Button variant="outline" className="mt-4" onClick={() => { void refetch() }}>
                            {t('common:actions.retry')}
                        </Button>
                    </div>
                ) : (
                    <>
                        <TransactionList
                            transactions={transactions}
                            grouped={list.params.sortBy === 'date'}
                            isLoading={isLoading}
                            emptyTitle={list.params.status === 'pending' ? t('transactions.emptyPendingTitle') : t('transactions.emptyTitle')}
                            emptyDescription={list.params.status === 'pending' ? t('transactions.emptyPendingDescription') : t('transactions.emptyDescription')}
                            emptyAction={
                                !isReadOnly ? (
                                    <Button onClick={handleCreate}>
                                        <Plus className="size-4" />
                                        {t('transactions.create')}
                                    </Button>
                                ) : undefined
                            }
                            onDelete={(id) => deleteTransaction.mutate(id)}
                            onDuplicate={(id) => {
                                const transaction = transactions.find((item) => item.id === id)
                                confirmIfNeeded(
                                    id,
                                    transaction ? warningsForConfirmedDuplicate(transaction) : [],
                                    (payload) => duplicateTransaction.mutate(payload),
                                )
                            }}
                            onConfirm={setApplying}
                            onSkip={(id) => skipTransaction.mutate(id)}
                            onEdit={openEdit}
                            isReadOnly={isReadOnly}
                        />

                        {loadedCount > 0 && (
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
                                        <Button variant="outline" size="sm" onClick={() => { void fetchNextPage() }}>
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
                                {!hasNextPage && !isFetchingNextPage && (meta?.last_page ?? 1) > 1 && (
                                    <p className="text-sm text-muted-foreground">{t('common:table.endOfList')}</p>
                                )}
                                <InfiniteScrollSentinel
                                    enabled={Boolean(hasNextPage) && !isFetchingNextPage && !isFetchNextPageError}
                                    onVisible={() => { void fetchNextPage() }}
                                />
                            </div>
                        )}
                    </>
                )}
            </div>
            </div>

            {negativeBalanceDialog}

            <ApplyDeferredDateDialog
                transaction={applying}
                open={applying !== null}
                onOpenChange={(open) => {
                    if (!open) {
                        setApplying(null)
                    }
                }}
                isSubmitting={confirmTransaction.isPending}
                onConfirm={(date) => {
                    if (!applying) {
                        return
                    }
                    confirmTransaction.mutate(
                        { id: applying.id, date },
                        { onSuccess: () => setApplying(null) },
                    )
                }}
            />
        </Page>
    )
}
