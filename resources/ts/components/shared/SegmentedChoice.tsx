import type { LucideIcon } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface SegmentedChoiceOption<T extends string> {
    value: T
    label: string
    icon?: LucideIcon
    color?: string
    /** A small square button: the icon alone, the label as its tooltip. */
    iconOnly?: boolean
}

interface SegmentedChoiceProps<T extends string> {
    value: T
    onChange: (value: T) => void
    options: readonly SegmentedChoiceOption<T>[]
    disabled?: boolean
}

export function SegmentedChoice<T extends string>({
    value,
    onChange,
    options,
    disabled,
}: SegmentedChoiceProps<T>) {
    return (
        <div className="flex gap-1 sm:gap-2 p-1 bg-muted rounded-lg">
            {options.map(({ value: option, label, icon: Icon, color, iconOnly }) => (
                <button
                    key={option}
                    type="button"
                    disabled={disabled}
                    onClick={() => onChange(option)}
                    title={iconOnly ? label : undefined}
                    aria-label={iconOnly ? label : undefined}
                    className={cn(
                        'flex items-center justify-center gap-1.5 sm:gap-2 py-2 rounded-md text-sm font-medium transition-all',
                        iconOnly ? 'aspect-square shrink-0 px-2' : 'min-w-0 flex-1 px-1.5 sm:px-4',
                        value === option
                            ? 'bg-background shadow-sm'
                            : 'hover:bg-background/50',
                        disabled && 'opacity-50 pointer-events-none'
                    )}
                >
                    {Icon && (
                        <Icon className={cn('size-4 shrink-0', value === option && color)} />
                    )}
                    {!iconOnly && <span className="truncate">{label}</span>}
                </button>
            ))}
        </div>
    )
}
