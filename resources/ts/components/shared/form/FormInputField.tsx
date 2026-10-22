import type { ComponentProps, ReactNode } from 'react'
import type { Control, FieldPath, FieldValues } from 'react-hook-form'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'

interface FormInputFieldProps<T extends FieldValues> {
    control: Control<T>
    name: FieldPath<T>
    label: ReactNode
    description?: ReactNode
    descriptionClassName?: string
    className?: string
    labelClassName?: string
    multiline?: boolean
    // Nullable values show as empty and an empty entry stores null (or a number for type="number").
    nullable?: boolean
    inputProps?: Omit<ComponentProps<'input'>, 'name' | 'value' | 'onChange' | 'onBlur' | 'ref'>
    textareaProps?: Omit<ComponentProps<'textarea'>, 'name' | 'value' | 'onChange' | 'onBlur' | 'ref'>
}

export function FormInputField<T extends FieldValues>({
    control,
    name,
    label,
    description,
    descriptionClassName,
    className,
    labelClassName,
    multiline,
    nullable,
    inputProps,
    textareaProps,
}: FormInputFieldProps<T>) {
    return (
        <FormField
            control={control}
            name={name}
            render={({ field }) => (
                <FormItem className={className}>
                    <FormLabel className={labelClassName}>{label}</FormLabel>
                    <FormControl>
                        {multiline ? (
                            <Textarea {...textareaProps} {...field} value={field.value ?? ''} />
                        ) : nullable ? (
                            <Input
                                {...inputProps}
                                {...field}
                                value={field.value ?? ''}
                                onChange={(e) => {
                                    const raw = e.target.value
                                    if (raw === '') return field.onChange(null)
                                    field.onChange(inputProps?.type === 'number' ? Number(raw) : raw)
                                }}
                            />
                        ) : (
                            <Input {...inputProps} {...field} />
                        )}
                    </FormControl>
                    {description && <FormDescription className={descriptionClassName}>{description}</FormDescription>}
                    <FormMessage />
                </FormItem>
            )}
        />
    )
}
