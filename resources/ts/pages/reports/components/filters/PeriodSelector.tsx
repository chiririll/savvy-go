import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { testIdSelect } from '@/lib/test-id'
import type { ReportFilters } from '../../types'
import { getMonthOptions, getQuarterOptions, getYearOptions } from '../../utils'

interface OptionSelectProps {
    value: string
    options: { value: string; label: string }[]
    onChange: (value: string) => void
    widthClass: string
}

function OptionSelect({ value, options, onChange, widthClass }: OptionSelectProps) {
    const { t } = useTranslation('pages')
    return (
        <Select value={value} onValueChange={onChange}>
            <SelectTrigger aria-label={t('reports.filters.period')} className={widthClass} {...testIdSelect('global-period-value')}>
                <SelectValue />
            </SelectTrigger>
            <SelectContent>
                {options.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                        {opt.label}
                    </SelectItem>
                ))}
            </SelectContent>
        </Select>
    )
}

interface PeriodSelectorProps {
    filters: ReportFilters
    onFilterChange: <K extends keyof ReportFilters>(key: K, value: ReportFilters[K]) => void
}

// The picker that goes with the chosen period type.
export function PeriodSelector({ filters, onFilterChange }: PeriodSelectorProps) {
    const { t, i18n } = useTranslation('pages')
    const monthOptions = useMemo(() => getMonthOptions(), [i18n.language])
    const quarterOptions = useMemo(() => getQuarterOptions(), [i18n.language])
    const yearOptions = useMemo(() => getYearOptions(), [])

    switch (filters.periodType) {
        case 'month':
            return (
                <OptionSelect
                    value={filters.selectedMonth}
                    options={monthOptions}
                    onChange={(val) => onFilterChange('selectedMonth', val)}
                    widthClass="w-full md:w-[180px]"
                />
            )
        case 'quarter':
            return (
                <OptionSelect
                    value={filters.selectedQuarter}
                    options={quarterOptions}
                    onChange={(val) => onFilterChange('selectedQuarter', val)}
                    widthClass="w-full md:w-[140px]"
                />
            )
        case 'year':
            return (
                <OptionSelect
                    value={filters.selectedYear}
                    options={yearOptions}
                    onChange={(val) => onFilterChange('selectedYear', val)}
                    widthClass="w-full md:w-[100px]"
                />
            )
        case 'custom':
            return (
                <div className="flex flex-col md:flex-row items-stretch md:items-center gap-2">
                    <Input
                        type="date"
                        aria-label={t('transactions.dateFrom')}
                        value={filters.customStartDate}
                        onChange={(e) => onFilterChange('customStartDate', e.target.value)}
                        className="w-full md:w-[140px]"
                    />
                    <span className="text-muted-foreground text-center hidden md:block">—</span>
                    <Input
                        type="date"
                        aria-label={t('transactions.dateTo')}
                        value={filters.customEndDate}
                        onChange={(e) => onFilterChange('customEndDate', e.target.value)}
                        className="w-full md:w-[140px]"
                    />
                </div>
            )
        default:
            return null
    }
}
