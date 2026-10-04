import { useFormContext } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Input } from '@/components/ui/input'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { useTransferRate } from './transfer-rate'

interface TransferRateFieldProps {
    rate: ReturnType<typeof useTransferRate>
    fromCode?: string
    toCode?: string
}

/** The transfer's exchange rate; read-only unless the currencies differ. */
export function TransferRateField({ rate, fromCode, toCode }: TransferRateFieldProps) {
    const { t } = useTranslation('forms')
    const { control } = useFormContext<TransactionFormValues>()

    return (
        <FormField
            control={control}
            name="exchange_rate"
            render={({ field }) => (
                <FormItem className="min-w-0">
                    <FormLabel>
                        {t('transactions.transferRate')}
                        {fromCode && toCode && (
                            <span className="text-muted-foreground ml-1">
                                ({fromCode} → {toCode})
                            </span>
                        )}
                    </FormLabel>
                    <FormControl>
                        <Input
                            type="number"
                            step="0.000001"
                            min={0}
                            placeholder="1"
                            {...field}
                            value={field.value ?? ''}
                            onChange={(event) => {
                                rate.startEditing()
                                field.onChange(event.target.value === '' ? null : Number(event.target.value))
                            }}
                            onFocus={rate.startEditing}
                            onBlur={() => {
                                field.onBlur()
                                rate.commit()
                            }}
                            onKeyDown={(event) => {
                                if (event.key === 'Enter') {
                                    event.preventDefault()
                                    event.currentTarget.blur()
                                }
                            }}
                            readOnly={!rate.editable}
                            disabled={!rate.editable}
                        />
                    </FormControl>
                    <FormMessage />
                </FormItem>
            )}
        />
    )
}
