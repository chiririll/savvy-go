import { CurrencySelect } from './CurrencySelect'

interface CurrencyIdFieldProps {
    value?: number | null
    onChange: (id: number) => void
    placeholder?: string
    disabled?: boolean
}

export function CurrencyIdField({
    value,
    onChange,
    placeholder,
    disabled,
}: CurrencyIdFieldProps) {
    return (
        <CurrencySelect
            value={value ? { source: 'existing', id: Number(value) } : null}
            onChange={(next) => {
                if (next.source === 'existing') {
                    onChange(next.id)
                }
            }}
            allowCatalog={false}
            placeholder={placeholder}
            disabled={disabled}
        />
    )
}
