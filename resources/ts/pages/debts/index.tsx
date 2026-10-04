import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { HandCoins, Banknote, TrendingDown, TrendingUp } from 'lucide-react'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import { DataTable, Page, PageHeader } from '@/components/shared'
import { createDebtColumns, DebtFormDialog, DebtPaymentDialog } from '@/components/features/debts'
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
import { formatCurrency } from '@/lib/utils'

export default function DebtsPage() {
    const { t } = useTranslation('pages')
    const [includeCompleted, setIncludeCompleted] = useState(false)
    const [paymentDialogOpen, setPaymentDialogOpen] = useState(false)
    const [selectedDebt, setSelectedDebt] = useState<Debt | null>(null)
    const [paymentMode, setPaymentMode] = useState<'payment' | 'collection'>('payment')

    const { data, isLoading } = useDebtsWithSummary({ include_completed: includeCompleted })
    const deleteDebt = useDeleteDebt()
    const createDebt = useCreateDebt()
    const updateDebt = useUpdateDebt()
    const debtPayment = useDebtPayment()
    const debtCollection = useDebtCollection()
    const reopenDebt = useReopenDebt()

    const debts = data?.data ?? []
    const summary = data?.summary
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

    const columns = createDebtColumns({
        onDelete: (id) => deleteDebt.mutate(id),
        onPayment: handlePayment,
        onCollect: handleCollect,
        onReopen: (id) => reopenDebt.mutate(id),
        onEdit: form.openEdit,
        isReadOnly,
    })

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
                <div className="grid gap-4 md:grid-cols-3">
                    <div className="rounded-lg border bg-card p-4">
                        <div className="flex items-center gap-2">
                            <div className="p-2 rounded-lg bg-red-100">
                                <TrendingDown className="size-5 text-red-600" />
                            </div>
                            <div>
                                <p className="text-sm text-muted-foreground">{t('debts.types.i_owe')}</p>
                                <p className="text-2xl font-bold text-red-600">
                                    {formatCurrency(summary.totalIOwe, summary.currency)}
                                </p>
                            </div>
                        </div>
                    </div>
                    <div className="rounded-lg border bg-card p-4">
                        <div className="flex items-center gap-2">
                            <div className="p-2 rounded-lg bg-green-100">
                                <TrendingUp className="size-5 text-green-600" />
                            </div>
                            <div>
                                <p className="text-sm text-muted-foreground">{t('debts.types.owed_to_me')}</p>
                                <p className="text-2xl font-bold text-green-600">
                                    {formatCurrency(summary.totalOwedToMe, summary.currency)}
                                </p>
                            </div>
                        </div>
                    </div>
                    <div className="rounded-lg border bg-card p-4">
                        <div className="flex items-center gap-2">
                            <div className={`p-2 rounded-lg ${summary.netDebt >= 0 ? 'bg-green-100' : 'bg-red-100'}`}>
                                {summary.netDebt >= 0 ? (
                                    <HandCoins className="size-5 text-green-600" />
                                ) : (
                                    <Banknote className="size-5 text-red-600" />
                                )}
                            </div>
                            <div>
                                <p className="text-sm text-muted-foreground">{t('debts.netPosition')}</p>
                                <p className={`text-2xl font-bold ${summary.netDebt >= 0 ? 'text-green-600' : 'text-red-600'}`}>
                                    {formatCurrency(Math.abs(summary.netDebt), summary.currency)}
                                    {' '}{summary.netDebt >= 0 ? t('debts.inYourFavor') : t('debts.youOwe')}
                                </p>
                            </div>
                        </div>
                    </div>
                </div>
            )}

            <div className="flex items-center space-x-2">
                <Switch
                    id="include-completed"
                    checked={includeCompleted}
                    onCheckedChange={setIncludeCompleted}
                />
                <Label htmlFor="include-completed">{t('debts.showCompleted')}</Label>
            </div>

            <DataTable
                columns={columns}
                data={debts}
                isLoading={isLoading}
            />

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
