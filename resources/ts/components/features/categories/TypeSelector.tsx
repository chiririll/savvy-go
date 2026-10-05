import { useTranslation } from 'react-i18next'
import { CATEGORY_TYPE_OPTIONS } from '@/constants/categories'
import { CategoryType } from '@/types'
import { SegmentedChoice } from '@/components/shared'
import { categoryTypeLabel } from '@/lib/labels'

interface TypeSelectorProps {
    value: CategoryType
    onChange: (value: CategoryType) => void
    error?: string
    disabled?: boolean
    disabledHelp?: string
}

export function TypeSelector({ value, onChange, error, disabled, disabledHelp }: TypeSelectorProps) {
    const { t } = useTranslation()

    return (
        <div className="space-y-2">
            <label className="text-sm font-medium">{t('fields.type')}</label>
            <SegmentedChoice
                value={value}
                onChange={onChange}
                disabled={disabled}
                options={CATEGORY_TYPE_OPTIONS.map((option) => ({
                    value: option.value,
                    label: categoryTypeLabel(t, option.value),
                    color: option.value === 'income' ? 'text-green-600' : 'text-red-600',
                }))}
            />
            {disabled && disabledHelp && (
                <p className="text-sm text-muted-foreground">{disabledHelp}</p>
            )}
            {error && <p className="text-sm text-destructive">{error}</p>}
        </div>
    )
}
