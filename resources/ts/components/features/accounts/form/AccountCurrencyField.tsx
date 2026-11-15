import { useTranslation } from 'react-i18next'
import { FormField, FormItem, FormLabel, FormMessage, FormDescription } from '@/components/ui/form'
import { AccountFormValues, encodeAccountCurrency } from '@/schemas'
import { CurrencySelect } from '@/components/shared/pickers/CurrencySelect'
import type { UseFormReturn } from 'react-hook-form'
import { toSelectValue } from './select-value'

interface AccountCurrencyFieldProps {
    form: UseFormReturn<AccountFormValues>
    /** The currency of an existing account cannot change. */
    isEdit?: boolean
}

export function AccountCurrencyField({ form, isEdit }: AccountCurrencyFieldProps) {
    const { t } = useTranslation(['common', 'forms'])
    const currency = form.watch('currency')

    return (
        <FormField
            control={form.control}
            name="currency"
            render={({ field }) => (
                <FormItem>
                    <FormLabel>{t('fields.currency')}</FormLabel>
                    <CurrencySelect
                        value={toSelectValue(field.value || currency)}
                        onChange={(next) => {
                            field.onChange(
                                encodeAccountCurrency(
                                    next.source === 'existing' ? { id: next.id } : { code: next.code },
                                ),
                            )
                        }}
                        placeholder={t('forms:selectCurrency')}
                        disabled={isEdit}
                    />
                    <FormDescription>
                        {isEdit ? t('forms:accounts.currencyLockedHelp') : t('forms:accounts.currencyCatalogHelp')}
                    </FormDescription>
                    <FormMessage />
                </FormItem>
            )}
        />
    )
}
