import type { Control, FieldPath, FieldValues } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { FormFooterCheckbox } from './FormFooterCheckbox'

interface FormActiveFieldProps<T extends FieldValues> {
    control: Control<T>
    name?: FieldPath<T>
    help: string
}

export function FormActiveField<T extends FieldValues>({
    control,
    name = 'is_active' as FieldPath<T>,
    help,
}: FormActiveFieldProps<T>) {
    const { t } = useTranslation('common')

    return <FormFooterCheckbox control={control} name={name} label={t('fields.active')} help={help} />
}
