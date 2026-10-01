import type { LucideIcon } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface SegmentedChoiceOption<T extends string> {
    value: T
    label: string
    icon?: LucideIcon
    color?: string
}

interface SegmentedChoiceProps<T extends string> {
    value: T
    onChange: (value: T) => void
    options: SegmentedChoiceOption<T>[]
    disabled?: boolean
}

export function SegmentedChoice<T extends string>({
    value,
    onChange,
    options,
    disabled,
}: SegmentedChoiceProps<T>) {
    return (
        <div className="flex gap-2 p-1 bg-muted rounded-lg">
            {options.map(({ value: option, label, icon: Icon, color }) => (
                <button
                    key={option}
                    type="button"
                    disabled={disabled}
                    onClick={() => onChange(option)}
                    className={cn(
                        'flex-1 flex items-center justify-center gap-2 py-2 px-4 rounded-md text-sm font-medium transition-all',
                        value === option
                            ? 'bg-background shadow-sm'
                            : 'hover:bg-background/50',
                        disabled && 'opacity-50 pointer-events-none'
                    )}
                >
                    {Icon && (
                        <Icon className={cn('size-4', value === option && color)} />
                    )}
                    {label}
                </button>
            ))}
        </div>
    )
}
