import type { Control, FieldPath, FieldValues } from 'react-hook-form'
import { Checkbox } from '@/components/ui/checkbox'
import { FormField } from '@/components/ui/form'
import { FieldHelp } from './FieldHelp'
import { FormDialogFooterStart } from './FormDialog'

interface FormFooterCheckboxProps<T extends FieldValues> {
    control: Control<T>
    name: FieldPath<T>
    label: string
    help: string
}

/** A labelled checkbox with a help hint, in the footer of a form dialog. */
export function FormFooterCheckbox<T extends FieldValues>({ control, name, label, help }: FormFooterCheckboxProps<T>) {
    return (
        <FormDialogFooterStart>
            <FormField
                control={control}
                name={name}
                render={({ field }) => (
                    <label className="flex items-center gap-2 text-sm font-normal leading-none">
                        <Checkbox checked={Boolean(field.value)} onCheckedChange={field.onChange} />
                        <span>{label}</span>
                        <FieldHelp>{help}</FieldHelp>
                    </label>
                )}
            />
        </FormDialogFooterStart>
    )
}
