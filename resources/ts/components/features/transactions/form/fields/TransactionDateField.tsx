import { useFormContext } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { FieldHelp } from '@/components/shared'
import type { TransactionFormValues } from '@/schemas/transactions'

interface TransactionDateFieldProps {
    /** Without a date the transaction stays pending; when required it can't be cleared. */
    required: boolean
    max?: string
}

export function TransactionDateField({ required, max }: TransactionDateFieldProps) {
    const { t } = useTranslation(['common', 'forms'])
    const { control } = useFormContext<TransactionFormValues>()

    return (
        <FormField
            control={control}
            name="date"
            render={({ field }) => (
                <FormItem className="min-w-0">
                    <FormLabel className="gap-1.5">
                        {t('fields.date')}
                        {!required && (
                            <FieldHelp>{t('forms:transactions.dateOptionalHelp')}</FieldHelp>
                        )}
                    </FormLabel>
                    <div className="flex items-center gap-1">
                        <FormControl>
                            <Input
                                type="date"
                                {...field}
                                value={field.value || ''}
                                required={required}
                                max={max}
                            />
                        </FormControl>
                        {!required && field.value && (
                            <Button
                                type="button"
                                variant="ghost"
                                size="icon-sm"
                                className="shrink-0 text-muted-foreground"
                                onClick={() => field.onChange('')}
                                aria-label={t('actions.clear')}
                                title={t('actions.clear')}
                            >
                                <X className="size-4" />
                            </Button>
                        )}
                    </div>
                    <FormMessage />
                </FormItem>
            )}
        />
    )
}
