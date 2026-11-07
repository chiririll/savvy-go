import type { Control, Path } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { FormControl, FormField, FormItem, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'

/** The title of a transaction: one line of plain text. Shared by the whole form and the restricted one. */
export function TransactionDescriptionField<T extends { description?: string }>({
    control,
    disabled,
}: {
    control: Control<T>
    disabled?: boolean
}) {
    const { t } = useTranslation(['common', 'forms'])

    return (
        <FormField
            control={control}
            name={'description' as Path<T>}
            render={({ field }) => (
                <FormItem>
                    <FormControl>
                        <Input
                            {...field}
                            value={(field.value as string | undefined) ?? ''}
                            autoFocus
                            maxLength={500}
                            disabled={disabled}
                            aria-label={t('fields.description')}
                            placeholder={t('forms:transactions.titlePlaceholder')}
                        />
                    </FormControl>
                    <FormMessage />
                </FormItem>
            )}
        />
    )
}
