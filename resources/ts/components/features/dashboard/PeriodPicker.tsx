import { useTranslation } from 'react-i18next'
import { Calendar } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { isValidDateRange, type PeriodPreset } from './period'

interface PeriodPickerProps {
    period: PeriodPreset
    onPeriodChange: (period: PeriodPreset) => void
    customStartDate: string
    customEndDate: string
    onCustomStartChange: (value: string) => void
    onCustomEndChange: (value: string) => void
}

export function PeriodPicker({
    period,
    onPeriodChange,
    customStartDate,
    customEndDate,
    onCustomStartChange,
    onCustomEndChange,
}: PeriodPickerProps) {
    const { t } = useTranslation('pages')
    const rangeInvalid = Boolean(customStartDate && customEndDate) && !isValidDateRange(customStartDate, customEndDate)

    return (
        <div className="flex flex-col gap-2 sm:items-end">
            <div className="flex items-center gap-2">
                <Calendar className="size-4 text-muted-foreground hidden sm:block" />
                <Select value={period} onValueChange={onPeriodChange}>
                    <SelectTrigger className="w-full sm:w-[200px] h-8" aria-label={t('dashboard.period')}>
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectItem value="last_30_days">{t('dashboard.periods.last_30_days')}</SelectItem>
                        <SelectItem value="this_month">{t('dashboard.periods.this_month')}</SelectItem>
                        <SelectItem value="last_month">{t('dashboard.periods.last_month')}</SelectItem>
                        <SelectItem value="last_3_months">{t('dashboard.periods.last_3_months')}</SelectItem>
                        <SelectItem value="last_6_months">{t('dashboard.periods.last_6_months')}</SelectItem>
                        <SelectItem value="this_year">{t('dashboard.periods.this_year')}</SelectItem>
                        <SelectItem value="custom">{t('dashboard.periods.custom')}</SelectItem>
                    </SelectContent>
                </Select>
            </div>
            {period === 'custom' && (
                <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-2">
                    <Input
                        type="date"
                        aria-label={t('transactions.dateFrom')}
                        value={customStartDate}
                        max={customEndDate || undefined}
                        aria-invalid={rangeInvalid}
                        onChange={(e) => onCustomStartChange(e.target.value)}
                        className="h-8 w-full sm:w-[140px]"
                    />
                    <span className="text-muted-foreground text-center hidden sm:block">—</span>
                    <Input
                        type="date"
                        aria-label={t('transactions.dateTo')}
                        value={customEndDate}
                        min={customStartDate || undefined}
                        aria-invalid={rangeInvalid}
                        onChange={(e) => onCustomEndChange(e.target.value)}
                        className="h-8 w-full sm:w-[140px]"
                    />
                </div>
            )}
            {period === 'custom' && rangeInvalid && (
                <p role="alert" className="text-xs text-destructive">
                    {t('dashboard.periodInvalid')}
                </p>
            )}
        </div>
    )
}
