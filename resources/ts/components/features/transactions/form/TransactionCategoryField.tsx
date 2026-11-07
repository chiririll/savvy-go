import type { Control } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { CategorySelect } from '@/components/shared'
import { FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import type { TransactionFormValues } from '@/schemas/transactions'

interface TransactionCategoryFieldProps {
    control: Control<TransactionFormValues>
    type: 'income' | 'expense'
}

/** The category of the transaction: what the payment was for, unless lines say it in parts. */
export function TransactionCategoryField({ control, type }: TransactionCategoryFieldProps) {
    const { t } = useTranslation('common')

    return (
        <FormField
            control={control}
            name="default_category_id"
            render={({ field }) => (
                <FormItem className="min-w-0">
                    <FormLabel>{t('fields.category')}</FormLabel>
                    <CategorySelect value={field.value} onChange={field.onChange} type={type} />
                    <FormMessage />
                </FormItem>
            )}
        />
    )
}
