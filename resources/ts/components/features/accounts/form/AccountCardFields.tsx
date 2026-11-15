import { useTranslation } from 'react-i18next'
import type { UseFormReturn } from 'react-hook-form'
import { FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import type { AccountFormValues } from '@/schemas'
import { CARD_GRACE_DAYS, CARD_STATEMENT_DAYS } from '@/types'

/** Terms of a credit card. One column on a phone, the two days side by side from `sm`. */
export function AccountCardFields({ form }: { form: UseFormReturn<AccountFormValues> }) {
    const { t } = useTranslation('forms')

    return (
        <section className="space-y-3">
            <h3 className="text-sm font-semibold">{t('accounts.card.title')}</h3>

            <FormField
                control={form.control}
                name="limit"
                render={({ field }) => (
                    <FormItem>
                        <FormLabel>{t('accounts.card.limit')}</FormLabel>
                        <FormControl>
                            <Input
                                type="number"
                                inputMode="decimal"
                                step="0.01"
                                min={0}
                                placeholder="0.00"
                                {...field}
                                value={field.value ?? ''}
                            />
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                )}
            />

            <div className="grid grid-cols-1 items-start gap-3 sm:grid-cols-2">
                <FormField
                    control={form.control}
                    name="statement_day"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>{t('accounts.card.statementDay')}</FormLabel>
                            <FormControl>
                                <Input
                                    type="number"
                                    inputMode="numeric"
                                    min={CARD_STATEMENT_DAYS.min}
                                    max={CARD_STATEMENT_DAYS.max}
                                    {...field}
                                    value={field.value ?? ''}
                                />
                            </FormControl>
                            <FormDescription>{t('accounts.card.statementDayHelp')}</FormDescription>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <FormField
                    control={form.control}
                    name="grace_days"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>{t('accounts.card.graceDays')}</FormLabel>
                            <FormControl>
                                <Input
                                    type="number"
                                    inputMode="numeric"
                                    min={CARD_GRACE_DAYS.min}
                                    max={CARD_GRACE_DAYS.max}
                                    {...field}
                                    value={field.value ?? ''}
                                />
                            </FormControl>
                            <FormDescription>{t('accounts.card.graceDaysHelp')}</FormDescription>
                            <FormMessage />
                        </FormItem>
                    )}
                />
            </div>
        </section>
    )
}
