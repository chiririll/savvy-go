import { useTranslation } from 'react-i18next'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { testIdSelect } from '@/lib/test-id'
import type { CompareType } from '../../types'

interface ComparisonSelectorProps {
    value: CompareType
    onChange: (value: CompareType) => void
}

export function ComparisonSelector({ value, onChange }: ComparisonSelectorProps) {
    const { t } = useTranslation('pages')

    return (
        <Select value={value} onValueChange={(val) => onChange(val as CompareType)}>
            <SelectTrigger aria-label={t('reports.filters.compareWith')} className="w-full md:w-[200px]" {...testIdSelect('global-compare')}>
                <SelectValue placeholder={t('reports.filters.comparePlaceholder')} />
            </SelectTrigger>
            <SelectContent>
                <SelectItem value="none">{t('reports.filters.noComparison')}</SelectItem>
                <SelectItem value="previous_period">{t('reports.filters.previousPeriod')}</SelectItem>
                <SelectItem value="same_period_last_year">{t('reports.filters.samePeriodLastYear')}</SelectItem>
            </SelectContent>
        </Select>
    )
}
