import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { schemaResolver } from '@/lib/form-resolver'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog'
import {
    Form,
    FormField,
    FormItem,
    FormLabel,
    FormControl,
    FormMessage,
    FormDescription,
} from '@/components/ui/form'
import { debtPaymentSchema, DebtPaymentFormData } from '@/schemas'
import { AccountSelect, useNegativeBalanceConfirm } from '@/components/shared'
import { useAccounts } from '@/hooks'
import { warningsForAccountOutflow } from '@/lib/negative-balance'
import { Debt } from '@/types'
import { formatCurrency, formatDateLocal } from '@/lib/utils'

interface DebtPaymentDialogProps {
    debt: Debt | null
    open: boolean
    onOpenChange: (open: boolean) => void
    onSubmit: (debtId: number, data: DebtPaymentFormData) => void
    isSubmitting?: boolean
    mode: 'payment' | 'collection'
}

export function DebtPaymentDialog({
    debt,
    open,
    onOpenChange,
    onSubmit,
    isSubmitting,
    mode,
}: DebtPaymentDialogProps) {
    const { t } = useTranslation(['forms', 'pages', 'common'])
    const { data: accounts } = useAccounts({ active: true, exclude_debts: true })
    const { confirmIfNeeded, dialog: negativeBalanceDialog } = useNegativeBalanceConfirm<DebtPaymentFormData>()

    const form = useForm<DebtPaymentFormData>({
        resolver: schemaResolver<DebtPaymentFormData>(debtPaymentSchema),
        defaultValues: {
            account_id: 0,
            amount: 0,
            date: formatDateLocal(),
            description: '',
        },
    })

    useEffect(() => {
        if (!open) {
            return
        }

        form.reset({
            account_id: 0,
            amount: 0,
            date: formatDateLocal(),
            description: '',
        })
    }, [open, debt?.id, mode, form])

    const handleSubmit = (data: DebtPaymentFormData) => {
        if (!debt) {
            return
        }

        const selectedAccount = accounts?.find((account) => account.id === Number(data.account_id))
        const warnings = mode === 'payment'
            ? warningsForAccountOutflow(selectedAccount, Number(data.amount) || 0)
            : []

        confirmIfNeeded(data, warnings, (payload) => onSubmit(debt.id, payload))
    }

    const title = mode === 'payment' ? t('debts.payment.makeTitle') : t('debts.payment.collectTitle')
    const description = mode === 'payment'
        ? t('debts.payment.makeDescription')
        : t('debts.payment.collectDescription')

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>{title}</DialogTitle>
                    <DialogDescription>{description}</DialogDescription>
                </DialogHeader>

                {debt && (
                    <div className="rounded-lg bg-muted p-4 space-y-2">
                        <div className="flex justify-between text-sm">
                            <span className="text-muted-foreground">{t('pages:debts.columns.debt')}</span>
                            <span className="font-medium">{debt.name}</span>
                        </div>
                        <div className="flex justify-between text-sm">
                            <span className="text-muted-foreground">{t('pages:debts.columns.remaining')}</span>
                            <span className="font-mono">{formatCurrency(debt.currentBalance, debt.currency)}</span>
                        </div>
                        {debt.counterparty && (
                            <div className="flex justify-between text-sm">
                                <span className="text-muted-foreground">
                                    {mode === 'payment' ? t('debts.payment.payTo') : t('debts.payment.receiveFrom')}
                                </span>
                                <span>{debt.counterparty}</span>
                            </div>
                        )}
                    </div>
                )}

                <Form {...form}>
                    <form onSubmit={form.handleSubmit(handleSubmit)} className="space-y-4">
                        <FormField
                            control={form.control}
                            name="account_id"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>{t('common:fields.account')}</FormLabel>
                                    <AccountSelect
                                        value={field.value}
                                        onChange={field.onChange}
                                        showBalance
                                    />
                                    <FormDescription>
                                        {mode === 'payment'
                                            ? t('debts.payment.accountPayFrom')
                                            : t('debts.payment.accountReceiveInto')
                                        }
                                    </FormDescription>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="amount"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>{t('common:fields.amount')}</FormLabel>
                                    <FormControl>
                                        <Input
                                            type="number"
                                            step="0.01"
                                            min={0}
                                            max={debt?.currentBalance}
                                            placeholder="0.00"
                                            {...field}
                                        />
                                    </FormControl>
                                    {debt && (
                                        <div className="flex gap-2">
                                            <Button
                                                type="button"
                                                variant="outline"
                                                size="sm"
                                                onClick={() => form.setValue('amount', debt.currentBalance)}
                                            >
                                                {t('debts.payment.fullAmount')}
                                            </Button>
                                            <Button
                                                type="button"
                                                variant="outline"
                                                size="sm"
                                                onClick={() => form.setValue('amount', Math.round(debt.currentBalance / 2 * 100) / 100)}
                                            >
                                                {t('debts.payment.half')}
                                            </Button>
                                        </div>
                                    )}
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="date"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>{t('common:fields.date')}</FormLabel>
                                    <FormControl>
                                        <Input type="date" max={formatDateLocal()} {...field} />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="description"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>{t('common:fields.description')}</FormLabel>
                                    <FormControl>
                                        <Textarea
                                            placeholder={t('debts.notesPlaceholder')}
                                            {...field}
                                        />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <DialogFooter>
                            <Button
                                type="button"
                                variant="outline"
                                onClick={() => onOpenChange(false)}
                            >
                                {t('common:actions.cancel')}
                            </Button>
                            <Button type="submit" disabled={isSubmitting}>
                                {isSubmitting
                                    ? t('common:actions.processing')
                                    : mode === 'payment' ? t('debts.payment.makeTitle') : t('debts.payment.collectTitle')
                                }
                            </Button>
                        </DialogFooter>
                    </form>
                </Form>
            </DialogContent>
            {negativeBalanceDialog}
        </Dialog>
    )
}
