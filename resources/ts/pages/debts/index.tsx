import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { HandCoins, Banknote, TrendingDown, TrendingUp, type LucideIcon } from 'lucide-react'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { FeedList, Page, PageHeader } from '@/components/shared'
import { DebtFormDialog, DebtPaymentDialog, DebtRow } from '@/components/features/debts'
import {
    useDebtsWithSummary,
    useDeleteDebt,
    useDebtPayment,
    useDebtCollection,
    useReopenDebt,
    useCreateDebt,
    useUpdateDebt,
    useResourceFormDialog,
} from '@/hooks'
import { useReadOnly } from '@/components/providers/ReadOnlyProvider'
import { Debt } from '@/types'
import { DebtFormData, DebtPaymentFormData } from '@/schemas'
import { cn, formatCurrency } from '@/lib/utils'

type DebtTab = 'active' | 'completed'

/** A summary figure: a row of one block on phones, a card from md up. */
function SummaryItem({ icon: Icon, tone, label, value, valueClassName }: {
    icon: LucideIcon
    tone: string
    label: string
    value: string
    valueClassName: string
}) {
    return (
        <div className="flex items-center gap-3 px-3 py-2.5 md:rounded-lg md:border md:bg-card md:p-4">
            <div className={cn('shrink-0 rounded-lg p-1.5 md:p-2', tone)}>
                <Icon className="size-4 md:size-5" />
            </div>
            <div className="flex min-w-0 flex-1 items-baseline justify-between gap-3 md:block">
                <p className="truncate text-sm text-muted-foreground">{label}</p>
                <p className={cn('shrink-0 text-right font-bold md:text-left md:text-2xl', valueClassName)}>{value}</p>
            </div>
        </div>
    )
}

export default function DebtsPage() {
    const { t } = useTranslation('pages')
    const [tab, setTab] = useState<DebtTab>('active')
    const [paymentDialogOpen, setPaymentDialogOpen] = useState(false)
    const [selectedDebt, setSelectedDebt] = useState<Debt | null>(null)
    const [paymentMode, setPaymentMode] = useState<'payment' | 'collection'>('payment')

    const { data, isLoading } = useDebtsWithSummary({ include_completed: tab === 'completed' })
    const deleteDebt = useDeleteDebt()
    const createDebt = useCreateDebt()
    const updateDebt = useUpdateDebt()
    const debtPayment = useDebtPayment()
    const debtCollection = useDebtCollection()
    const reopenDebt = useReopenDebt()

    const debts = data?.data ?? []
    const summary = data?.summary
    const visibleDebts = debts.filter((debt) => debt.isPaidOff === (tab === 'completed'))
    const isReadOnly = useReadOnly()
    const form = useResourceFormDialog<Debt, DebtFormData>({
        items: debts,
        isLoading,
        create: createDebt,
        update: updateDebt,
    })

    const handlePayment = (debt: Debt) => {
        setSelectedDebt(debt)
        setPaymentMode('payment')
        setPaymentDialogOpen(true)
    }

    const handleCollect = (debt: Debt) => {
        setSelectedDebt(debt)
        setPaymentMode('collection')
        setPaymentDialogOpen(true)
    }

    const handlePaymentSubmit = (debtId: number, formData: DebtPaymentFormData) => {
        if (paymentMode === 'payment') {
            debtPayment.mutate(
                { debtId, data: formData },
                { onSuccess: () => setPaymentDialogOpen(false) }
            )
        } else {
            debtCollection.mutate(
                { debtId, data: formData },
                { onSuccess: () => setPaymentDialogOpen(false) }
            )
        }
    }

    return (
        <Page title={t('debts.title')}>
            <div className="space-y-6">
            <PageHeader
                title={t('debts.title')}
                description={t('debts.description')}
                onCreateClick={isReadOnly ? undefined : form.openCreate}
                createLabel={t('debts.create')}
            />

            {summary && (
                <div className="divide-y rounded-lg border bg-card md:grid md:grid-cols-3 md:gap-4 md:divide-y-0 md:border-0 md:bg-transparent">
                    <SummaryItem
                        icon={TrendingDown}
                        tone="bg-red-100 text-red-600"
                        label={t('debts.types.i_owe')}
                        value={formatCurrency(summary.totalIOwe, summary.currency)}
                        valueClassName="text-red-600"
                    />
                    <SummaryItem
                        icon={TrendingUp}
                        tone="bg-green-100 text-green-600"
                        label={t('debts.types.owed_to_me')}
                        value={formatCurrency(summary.totalOwedToMe, summary.currency)}
                        valueClassName="text-green-600"
                    />
                    <SummaryItem
                        icon={summary.netDebt >= 0 ? HandCoins : Banknote}
                        tone={summary.netDebt >= 0 ? 'bg-green-100 text-green-600' : 'bg-red-100 text-red-600'}
                        label={t('debts.netPosition')}
                        value={`${formatCurrency(Math.abs(summary.netDebt), summary.currency)} ${summary.netDebt >= 0 ? t('debts.inYourFavor') : t('debts.youOwe')}`}
                        valueClassName={summary.netDebt >= 0 ? 'text-green-600' : 'text-red-600'}
                    />
                </div>
            )}

            <div className="mx-auto w-full max-w-[800px]">
                <Tabs value={tab} onValueChange={(value) => setTab(value as DebtTab)} className="mb-4">
                    <TabsList>
                        <TabsTrigger value="active">{t('debts.tabs.active')}</TabsTrigger>
                        <TabsTrigger value="completed">{t('debts.tabs.completed')}</TabsTrigger>
                    </TabsList>
                </Tabs>

                <FeedList
                    items={visibleDebts}
                    isLoading={isLoading}
                    emptyTitle={t('table.emptyTitle', { ns: 'common' })}
                    emptyDescription={t('table.emptyDescription', { ns: 'common' })}
                    onCreate={tab === 'active' ? form.openCreate : undefined}
                    createLabel={t('debts.create')}
                    isReadOnly={isReadOnly}
                    getKey={(debt) => debt.id}
                >
                    {(debt) => (
                        <DebtRow
                            debt={debt}
                            onEdit={form.openEdit}
                            onDelete={(id) => deleteDebt.mutate(id)}
                            onPayment={handlePayment}
                            onCollect={handleCollect}
                            onReopen={(id) => reopenDebt.mutate(id)}
                            isReadOnly={isReadOnly}
                        />
                    )}
                </FeedList>
            </div>

            <DebtFormDialog
                debt={form.entity}
                open={form.open}
                onOpenChange={form.setOpen}
                onSubmit={form.submit}
                isSubmitting={form.isSubmitting}
            />

            <DebtPaymentDialog
                debt={selectedDebt}
                open={paymentDialogOpen}
                onOpenChange={setPaymentDialogOpen}
                onSubmit={handlePaymentSubmit}
                isSubmitting={debtPayment.isPending || debtCollection.isPending}
                mode={paymentMode}
            />
            </div>
        </Page>
    )
}
