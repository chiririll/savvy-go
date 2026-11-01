import { useTranslation } from 'react-i18next'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { isAppLocale, LOCALE_LABELS, SUPPORTED_LOCALES, toAppLocale, type AppLocale } from '@/lib/i18n'

/** The language of the interface right now: what a language choice starts from. */
export function useUiLocale(): AppLocale {
    const { i18n } = useTranslation()
    return toAppLocale(i18n.resolvedLanguage ?? i18n.language)
}

interface Props {
    value: AppLocale
    onChange: (locale: AppLocale) => void
    className?: string
    disabled?: boolean
}

/** Picks the language of an email sent to someone else. It is not stored: it only styles that email. */
export function EmailLanguageSelect({ value, onChange, className, disabled }: Props) {
    const { t } = useTranslation('common')
    return (
        <Select
            value={value}
            disabled={disabled}
            onValueChange={(next) => {
                if (isAppLocale(next)) onChange(next)
            }}
        >
            <SelectTrigger className={className} aria-label={t('language.email')}>
                <SelectValue />
            </SelectTrigger>
            <SelectContent>
                {SUPPORTED_LOCALES.map((code) => (
                    <SelectItem key={code} value={code}>
                        {LOCALE_LABELS[code]}
                    </SelectItem>
                ))}
            </SelectContent>
        </Select>
    )
}
