import { useId } from 'react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { COLOR_OPTIONS } from '@/constants/categories'

interface ColorPickerProps {
    value: string
    onChange: (value: string) => void
    error?: string
}

export function ColorPicker({ value, onChange, error }: ColorPickerProps) {
    const { t } = useTranslation()
    const labelId = useId()

    return (
        <div className="space-y-2">
            <p id={labelId} className="text-sm font-medium">{t('fields.color')}</p>
            <div className="flex flex-wrap gap-2" role="group" aria-labelledby={labelId}>
                {COLOR_OPTIONS.map((color) => (
                    <button
                        key={color}
                        type="button"
                        className={cn(
                            'w-8 h-8 rounded-full border-2 transition-transform',
                            value === color
                                ? 'border-foreground scale-110'
                                : 'border-transparent'
                        )}
                        style={{ backgroundColor: color }}
                        onClick={() => onChange(color)}
                        aria-label={color}
                        aria-pressed={value === color}
                    />
                ))}
            </div>
            {error && <p className="text-sm text-destructive">{error}</p>}
        </div>
    )
}
