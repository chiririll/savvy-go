import type { Control, FieldPath, FieldValues } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { FormFooterCheckbox } from './FormFooterCheckbox'

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

    return <FormFooterCheckbox control={control} name={name} label={t('estimatedAmount')} help={help} />
}
