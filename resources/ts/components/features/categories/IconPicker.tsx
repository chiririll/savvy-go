import { useId } from 'react'
import { useTranslation } from 'react-i18next'
import { EmojiPicker } from '@/components/shared/pickers/EmojiPicker'

interface IconPickerProps {
    value: string
    onChange: (value: string) => void
    error?: string
}

export function IconPicker({ value, onChange, error }: IconPickerProps) {
    const { t } = useTranslation()
    const labelId = useId()

    return (
        <div className="space-y-2">
            <p id={labelId} className="text-sm font-medium">{t('fields.icon')}</p>
            <EmojiPicker value={value} onChange={onChange} labelledBy={labelId} />
            {error && <p className="text-sm text-destructive">{error}</p>}
        </div>
    )
}
