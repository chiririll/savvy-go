import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { formatDateLocal, isDateOverdue, parseDateKey } from '@/lib/dates'
import { intlLocale } from '@/lib/i18n'
import type { ApplyDateChoice } from './apply-deferred'
import { cn } from '@/lib/utils'

function formatStoredDate(date: string): string {
    return parseDateKey(date).toLocaleDateString(intlLocale())
}

interface ChoiceProps {
    name: string
    value: ApplyDateChoice
    checked: boolean
    onSelect: (choice: ApplyDateChoice) => void
    children: ReactNode
}

function Choice({ name, value, checked, onSelect, children }: ChoiceProps) {
    return (
        <label
            className={cn(
                'flex items-start gap-3 rounded-md border p-3 has-[:focus-visible]:ring-2',
                'has-[:focus-visible]:ring-ring',
            )}
        >
            <input
                type="radio"
                name={name}
                value={value}
                checked={checked}
                onChange={() => onSelect(value)}
                className="mt-1"
            />
            {children}
        </label>
    )
}

interface ApplyDateChoicesProps {
    groupId: string
    choice: ApplyDateChoice
    originalDate: string | null
    originalIsUsable: boolean
    customDate: string
    today: string
    onChoice: (choice: ApplyDateChoice) => void
    onCustomDate: (date: string) => void
}

/** Which date a deferred transaction takes when it is applied: today, its own, or another. */
export function ApplyDateChoices({
    groupId,
    choice,
    originalDate,
    originalIsUsable,
    customDate,
    today,
    onChoice,
    onCustomDate,
}: ApplyDateChoicesProps) {
    const { t } = useTranslation('pages')
    const { t: tCommon } = useTranslation('common')

    return (
        <fieldset className="grid gap-3">
            <legend className="sr-only">{t('transactions.applyDescription')}</legend>

            <Choice name={groupId} value="today" checked={choice === 'today'} onSelect={onChoice}>
                <span>
                    <span className="block text-sm font-medium">{t('transactions.applyToday')}</span>
                    <span className="text-xs text-muted-foreground">{formatStoredDate(formatDateLocal())}</span>
                </span>
            </Choice>

            {originalIsUsable && (
                <Choice name={groupId} value="original" checked={choice === 'original'} onSelect={onChoice}>
                    <span>
                        <span className="block text-sm font-medium">
                            {isDateOverdue(originalDate)
                                ? t('transactions.applyOverdue')
                                : t('transactions.applyOriginal')}
                        </span>
                        <span className="text-xs text-muted-foreground">
                            {originalDate ? formatStoredDate(originalDate) : null}
                        </span>
                    </span>
                </Choice>
            )}

            <Choice name={groupId} value="other" checked={choice === 'other'} onSelect={onChoice}>
                <span className="grid min-w-0 flex-1 gap-2">
                    <span className="text-sm font-medium">{t('transactions.applyOther')}</span>
                    {choice === 'other' && (
                        <div className="grid gap-1.5">
                            <Label htmlFor={`${groupId}-custom`}>{tCommon('fields.date')}</Label>
                            <Input
                                id={`${groupId}-custom`}
                                type="date"
                                value={customDate}
                                onChange={(event) => onCustomDate(event.target.value)}
                                max={today}
                                required
                            />
                        </div>
                    )}
                </span>
            </Choice>
        </fieldset>
    )
}
