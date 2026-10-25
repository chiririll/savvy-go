import type { ReactNode } from 'react'
import type { Control, FieldPath, FieldValues } from 'react-hook-form'
import { FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

export const NONE_VALUE = '__none__'

export interface ColumnOption {
    value: string
    label: string
}

interface SelectFieldProps<T extends FieldValues> {
    control: Control<T>
    name: FieldPath<T>
    label: string
    disabled: boolean
    help?: string
    placeholder?: string
    /** Value shown for the Select; the field's own value as a string. */
    toSelectValue: (value: unknown) => string
    onSelect: (value: string, onChange: (value: unknown) => void) => void
    children: ReactNode
}

function SelectField<T extends FieldValues>({
    control,
    name,
    label,
    disabled,
    help,
    placeholder,
    toSelectValue,
    onSelect,
    children,
}: SelectFieldProps<T>) {
    return (
        <FormField
            control={control}
            name={name}
            render={({ field }) => (
                <FormItem>
                    <FormLabel>{label}</FormLabel>
                    <Select
                        onValueChange={(val) => onSelect(val, field.onChange)}
                        value={toSelectValue(field.value)}
                        disabled={disabled}
                    >
                        <FormControl>
                            <SelectTrigger className="w-full">
                                <SelectValue placeholder={placeholder} />
                            </SelectTrigger>
                        </FormControl>
                        <SelectContent>{children}</SelectContent>
                    </Select>
                    {help && <FormDescription>{help}</FormDescription>}
                    <FormMessage />
                </FormItem>
            )}
        />
    )
}

interface ColumnFieldProps<T extends FieldValues> {
    control: Control<T>
    name: FieldPath<T>
    label: string
    options: ColumnOption[]
    disabled: boolean
    placeholder: string
    help?: string
    /** Label of the "no column" entry; without it the column is required. */
    noneLabel?: string
}

/** Picks the CSV column (by index) a value is read from. */
export function ColumnField<T extends FieldValues>({ options, placeholder, noneLabel, ...rest }: ColumnFieldProps<T>) {
    return (
        <SelectField
            {...rest}
            placeholder={placeholder}
            toSelectValue={(value) => (value as number | null | undefined)?.toString() ?? (noneLabel ? NONE_VALUE : '')}
            onSelect={(val, onChange) => onChange(val === NONE_VALUE ? null : Number(val))}
        >
            {noneLabel && <SelectItem value={NONE_VALUE}>{noneLabel}</SelectItem>}
            {options.map((opt) => (
                <SelectItem key={opt.value} value={opt.value}>
                    {opt.label}
                </SelectItem>
            ))}
        </SelectField>
    )
}

interface ChoiceFieldProps<T extends FieldValues> {
    control: Control<T>
    name: FieldPath<T>
    label: string
    disabled: boolean
    help?: string
    choices: { value: string; label: string }[]
}

/** Picks one of a few fixed values. */
export function ChoiceField<T extends FieldValues>({ choices, ...rest }: ChoiceFieldProps<T>) {
    return (
        <SelectField {...rest} toSelectValue={(value) => value as string} onSelect={(val, onChange) => onChange(val)}>
            {choices.map((choice) => (
                <SelectItem key={choice.value} value={choice.value}>
                    {choice.label}
                </SelectItem>
            ))}
        </SelectField>
    )
}
