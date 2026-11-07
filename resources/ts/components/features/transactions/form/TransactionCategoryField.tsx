import type { Control } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Plus, TriangleAlert } from 'lucide-react'
import { CategorySelect } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import type { TransactionFormValues } from '@/schemas/transactions'

interface TransactionCategoryFieldProps {
    control: Control<TransactionFormValues>
    type: 'income' | 'expense'
    /** The user wrote lines: this is the category of what they leave uncovered. */
    hasLines?: boolean
    /** The lines cover all of the payment, so there is nothing for this category. */
    locked?: boolean
    /** What the lines leave uncovered, as text, when it is not nothing: shown as a warning with the reason. */
    uncovered?: string | null
    /** Adds a line. */
    onAddLine?: () => void
}

/** The category of the transaction, or, with lines, the category of what the lines leave uncovered. */
export function TransactionCategoryField({
    control,
    type,
    hasLines,
    locked,
    uncovered,
    onAddLine,
}: TransactionCategoryFieldProps) {
    const { t } = useTranslation(['common', 'forms'])

    return (
        <FormField
            control={control}
            name="default_category_id"
            render={({ field }) => (
                <FormItem className="min-w-0">
                    <div className="flex items-center gap-1.5">
                        <FormLabel>
                            {hasLines ? t('forms:transactions.defaultCategory') : t('common:fields.category')}
                        </FormLabel>
                        {hasLines && uncovered && (
                            <Tooltip>
                                <TooltipTrigger asChild>
                                    <button
                                        type="button"
                                        aria-label={uncovered}
                                        className="inline-flex text-warning focus-visible:outline-2"
                                    >
                                        <TriangleAlert className="size-4" />
                                    </button>
                                </TooltipTrigger>
                                <TooltipContent>{uncovered}</TooltipContent>
                            </Tooltip>
                        )}
                    </div>
                    <div className="flex items-center gap-1">
                        <div className="min-w-0 flex-1">
                            <CategorySelect
                                value={field.value}
                                onChange={field.onChange}
                                type={type}
                                disabled={hasLines && locked}
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
