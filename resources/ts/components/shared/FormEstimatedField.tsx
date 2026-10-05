import type { Control, FieldPath, FieldValues } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Checkbox } from '@/components/ui/checkbox'
import { FormField } from '@/components/ui/form'
import { FieldHelp } from './FieldHelp'
import { FormDialogFooterStart } from './FormDialog'

interface FormEstimatedFieldProps<T extends FieldValues> {
    control: Control<T>
    help: string
    name?: FieldPath<T>
}

export function FormEstimatedField<T extends FieldValues>({
    control,
    help,
    name = 'is_estimated' as FieldPath<T>,
}: FormEstimatedFieldProps<T>) {
    const { t } = useTranslation('forms')

    return (
        <FormDialogFooterStart>
            <FormField
                control={control}
                name={name}
                render={({ field }) => (
                    <label className="flex items-center gap-2 text-sm font-normal leading-none">
                        <Checkbox
                            checked={Boolean(field.value)}
                            onCheckedChange={field.onChange}
                        />
                        <span>{t('estimatedAmount')}</span>
                        <FieldHelp>{help}</FieldHelp>
                    </label>
                )}
            />
        </FormDialogFooterStart>
    )
}
