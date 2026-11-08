import type { Control } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Plus } from 'lucide-react'
import { CategorySelect } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import type { TransactionFormValues } from '@/schemas/transactions'
import { testId } from '@/lib/test-id'

interface TransactionCategoryFieldProps {
    control: Control<TransactionFormValues>
    type: 'income' | 'expense'
    /** The user wrote lines: this is the category of what they leave uncovered. */
    hasLines?: boolean
    /** The lines cover all of the payment, so there is nothing for this category. */
    locked?: boolean
    /** Adds a line. */
    onAddLine?: () => void
}

/** The category of the transaction, or, with lines, the category of what the lines leave uncovered. */
export function TransactionCategoryField({
    control,
    type,
    hasLines,
    locked,
    onAddLine,
}: TransactionCategoryFieldProps) {
    const { t } = useTranslation(['common', 'forms'])

    return (
        <FormField
            control={control}
            name="default_category_id"
            render={({ field }) => (
                <FormItem className="min-w-0">
                    <FormLabel>
                        {hasLines ? t('forms:transactions.defaultCategory') : t('common:fields.category')}
                    </FormLabel>
                    <div className="flex items-center gap-1">
                        <div className="min-w-0 flex-1">
                            <CategorySelect
                                value={field.value}
                                onChange={field.onChange}
                                type={type}
                                disabled={hasLines && locked}
                                testName="default_category_id"
                            />
                        </div>
                        {onAddLine && (
                            <Button
                                type="button"
                                variant="ghost"
                                size="icon"
                                aria-label={t('forms:transactions.addLine')}
                                title={t('forms:transactions.addLine')}
                                onClick={onAddLine}
                                {...testId('tx-add-line')}
                            >
                                <Plus />
                            </Button>
                        )}
                    </div>
                    <FormMessage />
                </FormItem>
            )}
        />
    )
}
