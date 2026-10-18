import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/badge'
import { periodTypeLabel } from '@/lib/labels'
import { testIdControl, testIdControls } from '@/lib/test-id'
import type { PeriodType } from '../../types'

const QUICK_PERIODS: PeriodType[] = ['last_30_days', 'month', 'quarter', 'year', 'ytd']

interface PeriodTypeBadgesProps {
    value: PeriodType
    onChange: (type: PeriodType) => void
    scroll?: boolean
}

export function PeriodTypeBadges({ value, onChange, scroll = false }: PeriodTypeBadgesProps) {
    const { t } = useTranslation('pages')

    const badge = (type: PeriodType, label: string) => (
        <Badge
            key={type}
            {...testIdControl(type, value === type)}
            variant={value === type ? 'default' : 'outline'}
            className="cursor-pointer"
            onClick={() => onChange(type)}
        >
            {label}
        </Badge>
    )

    return (
        <div className={scroll ? 'flex w-max gap-1' : 'flex flex-wrap gap-1'} {...testIdControls('global-period')}>
            {QUICK_PERIODS.map((type) => badge(type, periodTypeLabel(t, type)))}
            {badge('custom', t('reports.filters.custom'))}
        </div>
    )
}
