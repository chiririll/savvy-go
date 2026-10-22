import type { UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { FormField, FormItem, FormLabel, FormControl, FormMessage, FormDescription } from '@/components/ui/form'
import { AccountSelect } from '@/components/shared/pickers/AccountSelect'
import type { DebtPaymentFormData } from '@/schemas'
import { Debt } from '@/types'
import { formatDateLocal } from '@/lib/utils'

interface DebtPaymentFieldsProps {
    form: UseFormReturn<DebtPaymentFormData>
    debt: Debt | null
    mode: 'payment' | 'collection'
}

export function DebtPaymentFields({ form, debt, mode }: DebtPaymentFieldsProps) {
    const { t } = useTranslation(['forms', 'pages', 'common'])

    return (
        <>
            <FormField
                control={form.control}
                name="account_id"
                render={({ field }) => (
                    <FormItem>
                        <FormLabel>{t('common:fields.account')}</FormLabel>
                        <AccountSelect value={field.value} onChange={field.onChange} showBalance />
                        <FormDescription>
                            {mode === 'payment'
                                ? t('debts.payment.accountPayFrom')
                                : t('debts.payment.accountReceiveInto')}
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
                                    onClick={() =>
                                        form.setValue('amount', Math.round((debt.currentBalance / 2) * 100) / 100)
                                    }
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
                            <Textarea placeholder={t('debts.notesPlaceholder')} {...field} />
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                )}
            />
        </>
    )
}
