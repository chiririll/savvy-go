import type { Control } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { CurrencyIdField, FormInputField } from '@/components/shared'
import { AccountSelect } from '@/components/shared/pickers/AccountSelect'
import { FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { formatCurrency } from '@/lib/utils'
import type { DebtFormData } from '@/schemas'
import type { Account } from '@/types'

interface DebtAmountFieldsProps {
    control: Control<DebtFormData>
    debtType: DebtFormData['debt_type']
    // The account the money moves through, or the currency picker when no transaction is created.
    showAccount: boolean
    showCurrency: boolean
    currencySymbol?: string
    insufficientFundsAccount?: Account
}

export function DebtAmountFields({
    control,
    debtType,
    showAccount,
    showCurrency,
    currencySymbol,
    insufficientFundsAccount,
}: DebtAmountFieldsProps) {
    const { t } = useTranslation(['common', 'forms'])

    return (
        <div className="grid grid-cols-2 gap-4">
            <FormInputField
                control={control}
                name="amount"
                className="min-w-0"
                label={(
                    <>
                        {t('fields.amount')}
                        {currencySymbol && <span className="text-muted-foreground ml-1">({currencySymbol})</span>}
                    </>
                )}
                inputProps={{ type: 'number', step: '0.01', min: 0, placeholder: '1000.00' }}
                description={insufficientFundsAccount
                    ? t('forms:debts.insufficientFunds', {
                        available: formatCurrency(
                            insufficientFundsAccount.currentBalance,
                            insufficientFundsAccount.currency,
                        ),
                    })
                    : t('forms:debts.amountHelp')}
                descriptionClassName={insufficientFundsAccount ? 'text-destructive' : undefined}
            />

            {showAccount && (
                <FormField
                    control={control}
                    name="account_id"
                    render={({ field }) => (
                        <FormItem className="min-w-0">
                            <FormLabel>
                                {debtType === 'i_owe'
                                    ? t('forms:debts.payment.accountReceiveInto')
                                    : t('forms:debts.payment.accountPayFrom')}
                            </FormLabel>
                            <AccountSelect value={field.value} onChange={field.onChange} />
                            <FormMessage />
                        </FormItem>
                    )}
                />
            )}

            {showCurrency && (
                <FormField
                    control={control}
                    name="currency_id"
                    render={({ field }) => (
                        <FormItem className="min-w-0">
                            <FormLabel>{t('fields.currency')}</FormLabel>
                            <CurrencyIdField
                                value={field.value}
                                onChange={field.onChange}
                                placeholder={t('forms:selectCurrency')}
                            />
                            <FormMessage />
                        </FormItem>
                    )}
                />
            )}
        </div>
    )
}
