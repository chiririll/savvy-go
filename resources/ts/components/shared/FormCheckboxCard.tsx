import type { ComponentProps, ReactNode } from 'react'
import type { Control, FieldPath, FieldValues } from 'react-hook-form'
import { Checkbox } from '@/components/ui/checkbox'
import { FormControl, FormDescription, FormField, FormItem, FormLabel } from '@/components/ui/form'

interface FormCheckboxCardProps<T extends FieldValues> {
    control: Control<T>
    name: FieldPath<T>
    label: ReactNode
    description?: ReactNode
    // Overrides the stored value when the box is forced on.
    forceChecked?: boolean
    disabled?: ComponentProps<typeof Checkbox>['disabled']
}

export function FormCheckboxCard<T extends FieldValues>({
    control,
    name,
    label,
    description,
    forceChecked,
    disabled,
}: FormCheckboxCardProps<T>) {
    return (
        <FormField
            control={control}
            name={name}
            render={({ field }) => (
                <FormItem className="flex flex-row items-start space-x-3 space-y-0 rounded-md border p-4">
                    <FormControl>
                        <Checkbox
                            checked={forceChecked || Boolean(field.value)}
                            disabled={disabled}
                            onCheckedChange={field.onChange}
                        />
                    </FormControl>
                    <div className="space-y-1 leading-none">
                        <FormLabel>{label}</FormLabel>
                        {description && <FormDescription>{description}</FormDescription>}
                    </div>
                </FormItem>
            )}
        />
    )
}
